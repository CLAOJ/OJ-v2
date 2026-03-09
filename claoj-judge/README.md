# CLAOJ Go Judge

Standalone Go judge for CLAOJ online judge system. Multiple instances can connect to a single backend.

## Quick Start

### Build

```bash
# Build locally
go build -o claoj-judge ./cmd/claoj-judge

# Build Docker image
docker build -t claoj/judge-go:latest .
```

### Run Locally

```bash
# Create config directory
mkdir -p /etc/claoj
cp judge.example.yml /etc/claoj/judge.yml

# Edit configuration
vi /etc/claoj/judge.yml

# Run judge
./claoj-judge \
  -server localhost \
  -port 9999 \
  -name MyJudge \
  -key YOUR_JUDGE_KEY
```

### Run with Docker

```bash
# Create judge network (if not exists)
docker network create claoj_judge

# Create problems volume (if not exists)
docker volume create claoj_problems

# Set environment variables
export JUDGE1_KEY="your-secure-key-1"
export JUDGE2_KEY="your-secure-key-2"

# Start all judges
docker-compose up -d
```

## Configuration

### Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-c` | Configuration file path | `/etc/claoj/judge.yml` |
| `-server` | Backend server host | (required) |
| `-port` | Backend server port | `9999` |
| `-name` | Judge name | (required) |
| `-key` | Judge authentication key | (required) |
| `-api-host` | API listen address | `0.0.0.0` |
| `-api-port` | API listen port | `9998` |
| `-log` | Log file path | stdout |
| `-no-watchdog` | Disable problem watcher | `false` |
| `-skip-self-test` | Skip executor self-tests | `false` |
| `-secure` | Use TLS connection | `false` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `CLAOJ_SERVER_HOST` | Backend server host |
| `CLAOJ_JUDGE_NAME` | Judge name |
| `CLAOJ_JUDGE_KEY` | Judge authentication key |

### Configuration File

See `judge.example.yml` for all available options.

## Architecture

```
                    ┌─────────────────┐
                    │   CLAOJ Backend │
                    │     (Go/Gin)    │
                    │   Port: 9999    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              │              │              │
    ┌─────────▼────┐ ┌──────▼──────┐ ┌────▼────────┐
    │   Judge 1    │ │   Judge 2   │ │   Judge N   │
    │  (Go Judge)  │ │  (Go Judge) │ │  (Go Judge) │
    │  Port: 9998  │ │  Port: 9998 │ │  Port: 9998 │
    └──────────────┘ └─────────────┘ └─────────────┘
```

## Supported Languages

| Language | Status | Notes |
|----------|--------|-------|
| C++17 | ✓ | Static linking |
| C++20 | ✓ | Static linking |
| C11 | ✓ | Static linking |
| Python 3 | ✓ | Requires python3 |
| Python 2 | ✓ | Requires python2 |
| Java 8 | ✓ | Requires OpenJDK |
| Go | ✓ | CGO disabled |
| Node.js | ✓ | Requires Node.js |
| Rust | ✓ | Static binaries |

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run benchmarks
go test -bench=. ./executors
```

### Adding a New Executor

1. Create new executor in `executors/<language>.go`:

```go
type MyExecutor struct {
    baseExecutor
}

func NewMyExecutor() *MyExecutor {
    return &MyExecutor{}
}

func (e *MyExecutor) Language() string {
    return "MYLANG"
}

func (e *MyExecutor) Compile(ctx context.Context, source string, dir string) (*CompileResult, error) {
    // Implementation
}

func (e *MyExecutor) Run(ctx context.Context, input string, limits RunLimits) (*RunResult, error) {
    // Implementation
}

func (e *MyExecutor) RuntimeVersions() []string {
    // Implementation
}
```

2. Register in `core/judge.go`:

```go
execList := []executors.Executor{
    // ... existing executors
    executors.NewMyExecutor(),
}
```

## Troubleshooting

### Judge Cannot Connect to Backend

1. Check network connectivity:
```bash
docker network inspect claoj_judge
```

2. Verify backend is running:
```bash
curl http://backend:9999/health
```

3. Check judge name and key in database:
```sql
SELECT * FROM judges WHERE name = 'Judge-Go-1';
```

### Compilation Fails

1. Verify compilers are installed in container:
```bash
docker exec claoj_judge_go_1 which g++ python3 java
```

2. Check compilation logs:
```bash
docker logs claoj_judge_go_1
```

### Permission Denied

1. Ensure SYS_PTRACE capability is added:
```yaml
cap_add:
  - SYS_PTRACE
```

2. Check volume permissions:
```bash
docker volume inspect claoj_problems
```

## License

Same as main CLAOJ project.
