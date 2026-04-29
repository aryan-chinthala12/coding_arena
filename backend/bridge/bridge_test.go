package bridge

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"testing"
	"time"
)

// fakeJudge simulates a DMOJ judge connecting to the bridge.
type fakeJudge struct {
	conn net.Conn
	t    *testing.T
}

func connectFakeJudge(t *testing.T, addr, id, key string) *fakeJudge {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to bridge: %v", err)
	}
	fj := &fakeJudge{conn: conn, t: t}

	// Send handshake
	fj.send(Packet{
		"name":      "handshake",
		"id":        id,
		"key":       key,
		"executors": map[string]any{"PY3": []any{"3.12.0"}, "CPP17": []any{"13.0"}},
		"problems":  []any{},
	})

	// Read handshake-success
	pkt := fj.recv()
	name, _ := pkt["name"].(string)
	if name != "handshake-success" {
		t.Fatalf("expected handshake-success, got %q", name)
	}

	return fj
}

func (fj *fakeJudge) send(pkt Packet) {
	fj.t.Helper()
	payload, err := json.Marshal(pkt)
	if err != nil {
		fj.t.Fatalf("marshal: %v", err)
	}

	var buf bytes.Buffer
	w, _ := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	w.Write(payload)
	w.Close()
	compressed := buf.Bytes()

	binary.Write(fj.conn, binary.BigEndian, uint32(len(compressed)))
	fj.conn.Write(compressed)
}

func (fj *fakeJudge) recv() Packet {
	fj.t.Helper()
	var size uint32
	if err := binary.Read(fj.conn, binary.BigEndian, &size); err != nil {
		fj.t.Fatalf("read size: %v", err)
	}
	compressed := make([]byte, size)
	if _, err := io.ReadFull(fj.conn, compressed); err != nil {
		fj.t.Fatalf("read payload: %v", err)
	}
	r, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		fj.t.Fatalf("zlib: %v", err)
	}
	defer r.Close()
	decompressed, _ := io.ReadAll(r)

	var pkt Packet
	if err := json.Unmarshal(decompressed, &pkt); err != nil {
		fj.t.Fatalf("unmarshal: %v", err)
	}
	return pkt
}

func (fj *fakeJudge) close() {
	fj.conn.Close()
}

// TestBridge_FullSubmission tests the complete flow:
// bridge starts → fake judge connects → submission sent → judge returns AC → result received.
func TestBridge_FullSubmission(t *testing.T) {
	b := New("127.0.0.1:0", "test-judge", "test-key")
	if err := b.Start(); err != nil {
		t.Fatalf("bridge start: %v", err)
	}
	defer b.Stop()

	addr := b.listener.Addr().String()

	// Connect fake judge
	fj := connectFakeJudge(t, addr, "test-judge", "test-key")
	defer fj.close()

	// Give the bridge a moment to register the judge
	time.Sleep(50 * time.Millisecond)

	if !b.HasJudge() {
		t.Fatal("expected judge to be connected")
	}

	// Submit in background
	type submitResult struct {
		result *SubmissionResult
		err    error
	}
	ch := make(chan submitResult, 1)
	go func() {
		r, err := b.Submit(context.Background(), "aplusb", "PY3", "print(sum(map(int,input().split())))", 2.0, 262144, false)
		ch <- submitResult{r, err}
	}()

	// Fake judge: read submission-request
	pkt := fj.recv()
	name, _ := pkt["name"].(string)
	if name != "submission-request" {
		t.Fatalf("expected submission-request, got %q", name)
	}
	subID := pkt["submission-id"]
	problemID, _ := pkt["problem-id"].(string)
	if problemID != "aplusb" {
		t.Fatalf("expected problem-id 'aplusb', got %q", problemID)
	}
	lang, _ := pkt["language"].(string)
	if lang != "PY3" {
		t.Fatalf("expected language PY3, got %q", lang)
	}

	// Fake judge: send acknowledgement
	fj.send(Packet{"name": "submission-acknowledged", "submission-id": subID})

	// Fake judge: send grading-begin
	fj.send(Packet{"name": "grading-begin", "submission-id": subID, "pretested": false})

	// Fake judge: send test case results (2 cases, both AC)
	fj.send(Packet{
		"name":          "test-case-status",
		"submission-id": subID,
		"cases": []any{
			map[string]any{
				"position":     1,
				"status":       0, // AC
				"time":         0.015,
				"points":       5.0,
				"total-points": 5.0,
				"memory":       4096,
				"output":       "3\n",
				"feedback":     "",
			},
			map[string]any{
				"position":     2,
				"status":       0, // AC
				"time":         0.012,
				"points":       5.0,
				"total-points": 5.0,
				"memory":       4096,
				"output":       "7\n",
				"feedback":     "",
			},
		},
	})

	// Fake judge: send grading-end
	fj.send(Packet{"name": "grading-end", "submission-id": subID})

	// Receive result
	select {
	case sr := <-ch:
		if sr.err != nil {
			t.Fatalf("submission error: %v", sr.err)
		}
		r := sr.result
		if r.Status != "AC" {
			t.Fatalf("expected AC, got %s", r.Status)
		}
		if len(r.Cases) != 2 {
			t.Fatalf("expected 2 cases, got %d", len(r.Cases))
		}
		if r.Points != 10.0 || r.TotalPoints != 10.0 {
			t.Fatalf("expected 10/10 points, got %.1f/%.1f", r.Points, r.TotalPoints)
		}
		if r.TotalTime < 0.02 {
			t.Fatalf("expected total time >= 0.02, got %.3f", r.TotalTime)
		}
		t.Logf("OK: status=%s points=%.0f/%.0f time=%.3fs cases=%d",
			r.Status, r.Points, r.TotalPoints, r.TotalTime, len(r.Cases))

	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
	}
}

// TestBridge_CompileError tests that compile errors are properly returned.
func TestBridge_CompileError(t *testing.T) {
	b := New("127.0.0.1:0", "test-judge", "test-key")
	if err := b.Start(); err != nil {
		t.Fatalf("bridge start: %v", err)
	}
	defer b.Stop()

	addr := b.listener.Addr().String()
	fj := connectFakeJudge(t, addr, "test-judge", "test-key")
	defer fj.close()
	time.Sleep(50 * time.Millisecond)

	ch := make(chan *SubmissionResult, 1)
	go func() {
		r, _ := b.Submit(context.Background(), "aplusb", "CPP17", "int main( {}", 2.0, 262144, false)
		ch <- r
	}()

	pkt := fj.recv()
	subID := pkt["submission-id"]

	// Judge sends compile-error
	fj.send(Packet{
		"name":          "compile-error",
		"submission-id": subID,
		"log":           "error: expected ')' before '{' token",
	})

	select {
	case r := <-ch:
		if r.Status != "CE" {
			t.Fatalf("expected CE, got %s", r.Status)
		}
		if r.CompileError == "" {
			t.Fatal("expected compile error message")
		}
		t.Logf("OK: status=%s error=%s", r.Status, r.CompileError)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// TestBridge_WrongAnswer tests mixed verdict (some AC, some WA).
func TestBridge_WrongAnswer(t *testing.T) {
	b := New("127.0.0.1:0", "test-judge", "test-key")
	if err := b.Start(); err != nil {
		t.Fatalf("bridge start: %v", err)
	}
	defer b.Stop()

	addr := b.listener.Addr().String()
	fj := connectFakeJudge(t, addr, "test-judge", "test-key")
	defer fj.close()
	time.Sleep(50 * time.Millisecond)

	ch := make(chan *SubmissionResult, 1)
	go func() {
		r, _ := b.Submit(context.Background(), "aplusb", "PY3", "print(0)", 2.0, 262144, false)
		ch <- r
	}()

	pkt := fj.recv()
	subID := pkt["submission-id"]

	fj.send(Packet{"name": "submission-acknowledged", "submission-id": subID})
	fj.send(Packet{"name": "grading-begin", "submission-id": subID, "pretested": false})
	fj.send(Packet{
		"name":          "test-case-status",
		"submission-id": subID,
		"cases": []any{
			map[string]any{"position": 1, "status": 0, "time": 0.01, "points": 5.0, "total-points": 5.0, "memory": 4096, "output": "", "feedback": ""},
			map[string]any{"position": 2, "status": 1, "time": 0.01, "points": 0.0, "total-points": 5.0, "memory": 4096, "output": "", "feedback": ""},
		},
	})
	fj.send(Packet{"name": "grading-end", "submission-id": subID})

	select {
	case r := <-ch:
		if r.Status != "WA" {
			t.Fatalf("expected WA, got %s", r.Status)
		}
		if r.Points != 5.0 || r.TotalPoints != 10.0 {
			t.Fatalf("expected 5/10, got %.1f/%.1f", r.Points, r.TotalPoints)
		}
		t.Logf("OK: status=%s points=%.0f/%.0f", r.Status, r.Points, r.TotalPoints)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out")
	}
}

// TestBridge_NoJudge tests that submit fails gracefully when no judge is connected.
func TestBridge_NoJudge(t *testing.T) {
	b := New("127.0.0.1:0", "test-judge", "test-key")
	if err := b.Start(); err != nil {
		t.Fatalf("bridge start: %v", err)
	}
	defer b.Stop()

	if b.HasJudge() {
		t.Fatal("expected no judges")
	}

	_, err := b.Submit(context.Background(), "aplusb", "PY3", "print(1)", 2.0, 262144, false)
	if err == nil {
		t.Fatal("expected error when no judge available")
	}
	t.Logf("OK: got expected error: %v", err)
}

// TestBridge_BadKey tests that judges with wrong keys are rejected.
func TestBridge_BadKey(t *testing.T) {
	b := New("127.0.0.1:0", "test-judge", "correct-key")
	if err := b.Start(); err != nil {
		t.Fatalf("bridge start: %v", err)
	}
	defer b.Stop()

	addr := b.listener.Addr().String()

	// Connect with wrong key
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	fj := &fakeJudge{conn: conn, t: t}
	fj.send(Packet{
		"name":      "handshake",
		"id":        "bad-judge",
		"key":       "wrong-key",
		"executors": map[string]any{},
		"problems":  []any{},
	})

	// Wait a moment — bridge should close the connection
	time.Sleep(100 * time.Millisecond)

	if b.HasJudge() {
		t.Fatal("judge with bad key should not be registered")
	}
	t.Log("OK: bad key rejected")
}

// TestStatusName tests the status code to string mapping.
func TestStatusName(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{StatusAC, "AC"},
		{StatusWA, "WA"},
		{StatusRTE, "RTE"},
		{StatusTLE, "TLE"},
		{StatusMLE, "MLE"},
		{StatusOLE, "OLE"},
		{StatusIR, "IR"},
		{StatusSC, "SC"},
		{StatusIE, "IE"},
		{StatusTLE | StatusWA, "TLE"}, // TLE takes priority
		{StatusMLE | StatusRTE, "MLE"},
	}

	for _, tt := range tests {
		got := StatusName(tt.code)
		if got != tt.want {
			t.Errorf("StatusName(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
