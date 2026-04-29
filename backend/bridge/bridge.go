package bridge

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

// Packet represents a generic DMOJ judge protocol packet.
type Packet map[string]any

// JudgeConn represents a connected DMOJ judge instance.
type JudgeConn struct {
	conn   net.Conn
	mu     sync.Mutex // serializes writes
	name   string
	closed atomic.Bool
}

// SubmissionResult holds the aggregated result of a graded submission.
type SubmissionResult struct {
	SubmissionID int64
	Status       string // "AC", "WA", "TLE", "MLE", "RTE", "CE", "IE", etc.
	CompileError string
	Cases        []CaseResult
	TotalTime    float64
	MaxMemory    int64
	Points       float64
	TotalPoints  float64
}

// CaseResult holds the result of a single test case.
type CaseResult struct {
	Position  int     `json:"position"`
	Status    int     `json:"status"`
	Time      float64 `json:"time"`
	Points    float64 `json:"points"`
	Total     float64 `json:"total-points"`
	Memory    int64   `json:"memory"`
	Output    string  `json:"output"`
	Feedback  string  `json:"feedback"`
}

// Status bitmask constants from DMOJ result.py.
const (
	StatusAC  = 0
	StatusWA  = 1 << 0
	StatusRTE = 1 << 1
	StatusTLE = 1 << 2
	StatusMLE = 1 << 3
	StatusIR  = 1 << 4
	StatusSC  = 1 << 5
	StatusOLE = 1 << 6
	StatusIE  = 1 << 30
)

// StatusName converts a DMOJ status bitmask to a human-readable string.
func StatusName(code int) string {
	if code == StatusAC {
		return "AC"
	}
	// Check from most severe to least
	if code&StatusIE != 0 {
		return "IE"
	}
	if code&StatusTLE != 0 {
		return "TLE"
	}
	if code&StatusMLE != 0 {
		return "MLE"
	}
	if code&StatusOLE != 0 {
		return "OLE"
	}
	if code&StatusRTE != 0 {
		return "RTE"
	}
	if code&StatusIR != 0 {
		return "IR"
	}
	if code&StatusWA != 0 {
		return "WA"
	}
	if code&StatusSC != 0 {
		return "SC"
	}
	return fmt.Sprintf("UNKNOWN(%d)", code)
}

// Bridge is a TCP server that speaks the DMOJ judge wire protocol.
// Judges connect to the bridge; the bridge dispatches submissions and collects results.
type Bridge struct {
	listener net.Listener
	addr     string
	judgeID  string
	judgeKey string

	mu     sync.Mutex
	judges map[string]*JudgeConn // name -> connection

	// Pending submissions awaiting results.
	pending   sync.Map // submissionID (int64) -> chan *SubmissionResult
	nextID    atomic.Int64
	closeOnce sync.Once
	done      chan struct{}
}

// New creates a new Bridge that listens on the given address.
// judgeID and judgeKey are used to authenticate connecting judges.
func New(addr, judgeID, judgeKey string) *Bridge {
	b := &Bridge{
		addr:     addr,
		judgeID:  judgeID,
		judgeKey: judgeKey,
		judges:   make(map[string]*JudgeConn),
		done:     make(chan struct{}),
	}
	b.nextID.Store(1)
	return b
}

// Start begins listening for judge connections in the background.
func (b *Bridge) Start() error {
	ln, err := net.Listen("tcp", b.addr)
	if err != nil {
		return fmt.Errorf("bridge listen: %w", err)
	}
	b.listener = ln
	log.Printf("[BRIDGE] Listening on %s for judge connections", b.addr)

	go b.acceptLoop()
	return nil
}

// Stop gracefully shuts down the bridge.
func (b *Bridge) Stop() {
	b.closeOnce.Do(func() {
		close(b.done)
		if b.listener != nil {
			b.listener.Close()
		}
		b.mu.Lock()
		for _, jc := range b.judges {
			jc.close()
		}
		b.mu.Unlock()
	})
}

// HasJudge returns true if at least one judge is connected and available.
func (b *Bridge) HasJudge() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.judges) > 0
}

// Submit sends a submission to an available judge and waits for the result.
// It blocks until the judge finishes grading or the context expires.
func (b *Bridge) Submit(ctx context.Context, problemID, language, source string, timeLimit float64, memoryLimit int64, shortCircuit bool) (*SubmissionResult, error) {
	judge := b.pickJudge()
	if judge == nil {
		return nil, fmt.Errorf("no judge available")
	}

	subID := b.nextID.Add(1) - 1

	// Create result channel
	resultCh := make(chan *SubmissionResult, 1)
	b.pending.Store(subID, resultCh)
	defer b.pending.Delete(subID)

	// Send submission-request
	pkt := Packet{
		"name":          "submission-request",
		"submission-id": subID,
		"problem-id":    problemID,
		"language":      language,
		"source":        source,
		"time-limit":    timeLimit,
		"memory-limit":  memoryLimit,
		"short-circuit": shortCircuit,
		"meta":          map[string]any{},
	}

	if err := judge.send(pkt); err != nil {
		return nil, fmt.Errorf("send submission: %w", err)
	}

	log.Printf("[BRIDGE] Sent submission %d to judge %q (problem=%s, lang=%s)",
		subID, judge.name, problemID, language)

	// Wait for result or context cancellation
	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.done:
		return nil, fmt.Errorf("bridge shutting down")
	}
}

// --- internal ---

func (b *Bridge) acceptLoop() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.done:
				return
			default:
				log.Printf("[BRIDGE] Accept error: %v", err)
				continue
			}
		}
		go b.handleJudge(conn)
	}
}

func (b *Bridge) handleJudge(conn net.Conn) {
	jc := &JudgeConn{conn: conn}
	defer func() {
		jc.close()
		b.mu.Lock()
		if jc.name != "" {
			delete(b.judges, jc.name)
		}
		b.mu.Unlock()
		log.Printf("[BRIDGE] Judge %q disconnected", jc.name)
	}()

	// Read handshake
	pkt, err := jc.recv()
	if err != nil {
		log.Printf("[BRIDGE] Handshake read error: %v", err)
		return
	}

	name, _ := pkt["name"].(string)
	if name != "handshake" {
		log.Printf("[BRIDGE] Expected handshake, got %q", name)
		return
	}

	judgeID, _ := pkt["id"].(string)
	judgeKey, _ := pkt["key"].(string)

	if judgeKey != b.judgeKey {
		log.Printf("[BRIDGE] Judge %q: invalid key", judgeID)
		return
	}

	jc.name = judgeID

	// Send handshake-success
	if err := jc.send(Packet{"name": "handshake-success"}); err != nil {
		log.Printf("[BRIDGE] Handshake response error: %v", err)
		return
	}

	log.Printf("[BRIDGE] Judge %q connected from %s", judgeID, conn.RemoteAddr())

	b.mu.Lock()
	b.judges[judgeID] = jc
	b.mu.Unlock()

	// Read packets from judge forever
	for {
		pkt, err := jc.recv()
		if err != nil {
			if !jc.closed.Load() {
				log.Printf("[BRIDGE] Judge %q read error: %v", judgeID, err)
			}
			return
		}
		b.handlePacket(jc, pkt)
	}
}

func (b *Bridge) handlePacket(jc *JudgeConn, pkt Packet) {
	name, _ := pkt["name"].(string)
	subID := jsonInt64(pkt, "submission-id")

	switch name {
	case "ping-response":
		// ignore

	case "submission-acknowledged":
		log.Printf("[BRIDGE] Judge %q acknowledged submission %d", jc.name, subID)

	case "grading-begin":
		log.Printf("[BRIDGE] Grading begun for submission %d", subID)

	case "compile-error":
		compileLog, _ := pkt["log"].(string)
		result := &SubmissionResult{
			SubmissionID: subID,
			Status:       "CE",
			CompileError: compileLog,
		}
		b.deliverResult(subID, result)

	case "compile-message":
		compileLog, _ := pkt["log"].(string)
		log.Printf("[BRIDGE] Compile message for %d: %s", subID, compileLog)

	case "test-case-status":
		// Accumulate — we handle final result at grading-end
		casesRaw, ok := pkt["cases"].([]any)
		if !ok {
			return
		}
		for _, c := range casesRaw {
			caseMap, ok := c.(map[string]any)
			if !ok {
				continue
			}
			cr := CaseResult{
				Position: int(jsonFloat64(caseMap, "position")),
				Status:   int(jsonFloat64(caseMap, "status")),
				Time:     jsonFloat64(caseMap, "time"),
				Points:   jsonFloat64(caseMap, "points"),
				Total:    jsonFloat64(caseMap, "total-points"),
				Memory:   int64(jsonFloat64(caseMap, "memory")),
			}
			if out, ok := caseMap["output"].(string); ok {
				cr.Output = out
			}
			if fb, ok := caseMap["feedback"].(string); ok {
				cr.Feedback = fb
			}

			// Store case in pending accumulator
			b.accumulateCase(subID, cr)
		}

	case "grading-end":
		b.finalizeResult(subID)

	case "batch-begin", "batch-end":
		// informational, no action needed

	case "internal-error":
		msg, _ := pkt["message"].(string)
		log.Printf("[BRIDGE] Internal error for %d: %s", subID, msg)
		result := &SubmissionResult{
			SubmissionID: subID,
			Status:       "IE",
			CompileError: msg,
		}
		b.deliverResult(subID, result)

	case "submission-terminated":
		log.Printf("[BRIDGE] Submission %d terminated", subID)
		result := &SubmissionResult{
			SubmissionID: subID,
			Status:       "AB", // aborted
		}
		b.deliverResult(subID, result)

	default:
		log.Printf("[BRIDGE] Unknown packet from %q: %s", jc.name, name)
	}
}

// caseAccumulator stores in-progress test case results per submission.
var caseAccumulators sync.Map // int64 -> *[]CaseResult

func (b *Bridge) accumulateCase(subID int64, cr CaseResult) {
	val, _ := caseAccumulators.LoadOrStore(subID, &[]CaseResult{})
	cases := val.(*[]CaseResult)
	*cases = append(*cases, cr)
}

func (b *Bridge) finalizeResult(subID int64) {
	val, ok := caseAccumulators.LoadAndDelete(subID)
	cases := &[]CaseResult{}
	if ok {
		cases = val.(*[]CaseResult)
	}

	result := &SubmissionResult{
		SubmissionID: subID,
		Cases:        *cases,
	}

	// Aggregate stats
	worstStatus := 0
	for _, c := range *cases {
		result.TotalTime += c.Time
		result.Points += c.Points
		result.TotalPoints += c.Total
		if c.Memory > result.MaxMemory {
			result.MaxMemory = c.Memory
		}
		worstStatus |= c.Status
	}

	result.Status = StatusName(worstStatus)
	b.deliverResult(subID, result)
}

func (b *Bridge) deliverResult(subID int64, result *SubmissionResult) {
	val, ok := b.pending.Load(subID)
	if !ok {
		log.Printf("[BRIDGE] No pending channel for submission %d", subID)
		return
	}
	ch := val.(chan *SubmissionResult)
	select {
	case ch <- result:
	default:
		log.Printf("[BRIDGE] Result channel full for submission %d", subID)
	}
}

func (b *Bridge) pickJudge() *JudgeConn {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, jc := range b.judges {
		if !jc.closed.Load() {
			return jc
		}
	}
	return nil
}

// --- JudgeConn I/O ---

func (jc *JudgeConn) send(pkt Packet) error {
	payload, err := json.Marshal(pkt)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return fmt.Errorf("zlib writer: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("zlib write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("zlib close: %w", err)
	}

	compressed := buf.Bytes()

	jc.mu.Lock()
	defer jc.mu.Unlock()

	// Write 4-byte big-endian length + compressed data
	if err := binary.Write(jc.conn, binary.BigEndian, uint32(len(compressed))); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := jc.conn.Write(compressed); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

func (jc *JudgeConn) recv() (Packet, error) {
	// Read 4-byte big-endian length
	var size uint32
	if err := binary.Read(jc.conn, binary.BigEndian, &size); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	// Sanity check: reject packets > 16 MB
	if size > 16*1024*1024 {
		return nil, fmt.Errorf("packet too large: %d bytes", size)
	}

	// Read compressed payload
	compressed := make([]byte, size)
	if _, err := io.ReadFull(jc.conn, compressed); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("zlib reader: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("zlib read: %w", err)
	}

	// Parse JSON
	var pkt Packet
	if err := json.Unmarshal(decompressed, &pkt); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	return pkt, nil
}

func (jc *JudgeConn) close() {
	if jc.closed.CompareAndSwap(false, true) {
		jc.conn.Close()
	}
}

// jsonFloat64 safely extracts a float64 from a map (JSON numbers decode as float64).
func jsonFloat64(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// jsonInt64 extracts an int64 from a Packet (JSON numbers are float64).
func jsonInt64(pkt Packet, key string) int64 {
	return int64(jsonFloat64(pkt, key))
}
