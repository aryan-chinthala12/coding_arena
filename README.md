# Coding Arena

Self-hosted competitive programming judge built for GCET's open source program. Submit code, get it graded against test cases in a sandboxed environment. Runs entirely in Docker with no external dependencies.

## Setup

```bash
git clone https://github.com/GCET-Open-Source-Foundation/coding_arena.git
cd coding_arena
docker compose up --build
```

First build takes a while (judge image compiles Rust toolchain, fetches Kotlin/Dart SDKs). Subsequent builds are cached.

Once all three containers are up:
- http://localhost — frontend
- http://localhost:8080/health — `{"status":"ok","judge":"connected"}`

Smoke test the full pipeline:

```bash
./scripts/test-judge.sh
```

## How it works

Three containers:

- **Frontend** — React + TypeScript + Vite, served via nginx on port 80. Monaco editor, problem browser, submission results.
- **Backend** — Go (Gin) on port 8080. Accepts submissions, manages a TCP bridge on port 9999 that the judge connects to.
- **Judge** — DMOJ judge-server in a custom `debian:trixie-slim` image (`arena`). Connects outbound to the backend bridge, receives submissions, runs them sandboxed, returns verdicts.

No database. Problems are flat files (`init.yml` + `.in`/`.out` pairs) mounted into the judge container from `judge-config/problems/`.

## Project layout

```
backend/               Go API + bridge (see backend/ for details)
frontend/              React SPA (see frontend/README.md)
judge-server/          DMOJ source + custom Dockerfile at .docker/arena/
judge-config/          .dmojrc + problems/
scripts/               test-judge.sh (e2e smoke test)
docker-compose.yml
```

## Adding problems

Create `judge-config/problems/<slug>/`:

```
init.yml
1.in   1.out
2.in   2.out
```

```yaml
test_cases:
- {in: 1.in, out: 1.out, points: 5}
- {in: 2.in, out: 2.out, points: 5}

time_limit: 1.0
memory_limit: 262144
```

Restart the judge to pick them up. Output comparison is byte-exact, so trailing newlines matter.

## Configuration

Environment variables on the backend (defaults in `docker-compose.yml`):

- `PORT` — HTTP port (8080)
- `BRIDGE_ADDR` — bridge listen address (:9999)
- `JUDGE_ID` / `JUDGE_KEY` — judge auth credentials
- `API_KEYS` — comma-separated API keys, auth disabled if unset
- `CORS_ORIGINS` — allowed origins
- `JUDGE_TIME_LIMIT` — per-case time override (Go duration, e.g. `5s`)
- `JUDGE_MEMORY_LIMIT` — memory override in MB

## Development

```bash
cd backend && go test ./...          # backend tests
cd frontend && npm run dev           # frontend dev server on :5173
```

## Contributing

Fork, branch, PR against `main` on [Aerosane/coding_arena](https://github.com/Aerosane/coding_arena), not the GCET org repo. Commits follow `type(scope): description`. Backend changes should pass `go test ./...`. Problem PRs should include properly formatted test data (look at existing problems for reference).

## License

MIT
