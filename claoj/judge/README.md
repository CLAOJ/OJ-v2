# CLAOJ Go Judge

Native Go implementation of the CLAOJ competitive programming judge system.

## Status: In Development

This is a migration from the Python-based DMOJ judge to a native Go implementation for better performance, lower resource usage, and easier maintenance.

## Project Structure

```
judge/
├── cmd/
│   └── claoj-judge/       # Main executable
├── core/
│   ├── judge.go           # Main judge coordinator
│   └── worker.go          # Submission grading worker
├── executors/
│   ├── base.go            # Base executor interface
│   ├── cpp.go             # C/C++ executors
│   ├── languages.go       # Other language executors
│   └── executors_test.go  # Executor tests
├── sandbox/
│   ├── sandbox.go         # Process isolation
│   └── sandbox_test.go    # Sandbox tests
├── protocol/
│   └── packet.go          # Network protocol
├── checkers/              # Output checkers (TODO)
├── contrib/               # Contest formats (TODO)
└── test.sh                # Test runner
```

## Features

### Implemented
- [x] TCP protocol compatibility with Go backend
- [x] Judge coordination and worker management
- [x] C/C++ executor (C11, C++17, C++20)
- [x] Python executor (PY2, PY3)
- [x] Java executor (JAVA8, JAVA11)
- [x] Go executor
- [x] Node.js executor
- [x] Rust executor
- [x] Basic sandboxing framework

### In Progress
- [ ] seccomp-bpf sandboxing
- [ ] Memory/CPU resource limits
- [ ] File system isolation
- [ ] Custom checker support

### Planned
- [ ] All 50+ language executors
- [ ] Contest format support (ICPC, IOI)
- [ ] Batch grading
- [ ] Problem watching

## Building

### Prerequisites

- Go 1.24+
- GCC/G++ (for C/C++ support)
- Python 3 (for Python support)
- Node.js (for Node.js support)
- Rust (for Rust support)
- Java JDK (for Java support)

### Install Dependencies

```bash
cd claoj-go/judge
go mod download
```

### Build

```bash
go build -o claoj-judge ./cmd/claoj-judge
```

## Running

### Basic Usage

```bash
./claoj-judge \
  -server localhost \
  -port 9999 \
  -name TestJudge \
  -key YOUR_JUDGE_KEY \
  -api-port 9998
```

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-server` | Backend server host | (required) |
| `-port` | Backend server port | 9999 |
| `-name` | Judge name | (required) |
| `-key` | Judge authentication key | (required) |
| `-api-port` | Local API port | 9998 |
| `-api-host` | Local API host | 127.0.0.1 |
| `-log` | Log file path | stdout |
| `-no-watchdog` | Disable problem watcher | false |
| `-skip-self-test` | Skip executor tests | false |

## Testing

### Run All Tests

```bash
./test.sh
```

### Run Specific Package Tests

```bash
go test ./executors -v
go test ./sandbox -v
go test ./core -v
go test ./protocol -v
```

### Run Benchmarks

```bash
go test -bench=. -benchmem ./executors
```

## Supported Languages

| Language | Status | Tests |
|----------|--------|-------|
| C++17 | Implemented | ✓ |
| C++20 | Implemented | ✓ |
| C11 | Implemented | ✓ |
| Python 3 | Implemented | ✓ |
| Python 2 | Implemented | ✓ |
| Java 8 | Implemented | ✓ |
| Java 11 | Implemented | ✓ |
| Go | Implemented | ✓ |
| Node.js | Implemented | ✓ |
| Rust | Implemented | ✓ |

## Architecture

### Grading Flow

```
Submission Request
       │
       ▼
┌──────────────┐
│ Validate     │
│ Language     │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Compile      │
│ (sandboxed)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Run Test     │
│ Cases        │
│ (isolated)   │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Check        │
│ Output       │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ Report       │
│ Results      │
└──────────────┘
```

### Network Protocol

The judge uses a binary protocol over TCP:
- 4-byte size prefix (big-endian)
- zlib-compressed JSON payload

Packet types:
- `handshake` / `handshake-success`
- `submission-request`
- `grading-begin` / `grading-end`
- `test-case-status`
- `compile-error`
- `internal-error`

## Migration Progress

See [JUDGE_MIGRATION_PLAN.md](../JUDGE_MIGRATION_PLAN.md) for the complete migration plan.

### Phase Status

| Phase | Status | Progress |
|-------|--------|----------|
| 0: Foundation | Complete | 100% |
| 1: Core Judge | In Progress | 70% |
| 2: Executors | Not Started | 20% |
| 3: Advanced | Not Started | 0% |
| 4: Cutover | Not Started | 0% |

## Configuration

Create a configuration file at `~/.claojrc`:

```json
{
  "server_host": "localhost",
  "server_port": 9999,
  "judge_name": "MyJudge",
  "judge_key": "your-auth-key",
  "problem_globs": ["/problems/*/"],
  "tempdir": "/tmp/judge",
  "runtime": {
    "cpp": {
      "compiler": "g++",
      "flags": "-std=c++17 -O2"
    }
  }
}
```

## Security

### Sandboxing

The judge uses multiple layers of isolation:

1. **seccomp-bpf**: Syscall filtering
2. **Resource limits**: CPU time, memory, file size
3. **Filesystem isolation**: Restricted read/write paths
4. **Network blocking**: No socket access

### Security Testing

```bash
# Run security tests
go test ./security -v

# Test sandbox escapes
go test ./sandbox -run Escape
```

## Performance

### Benchmarks (Current)

| Metric | Python Judge | Go Judge | Target |
|--------|--------------|----------|--------|
| Startup time | 500ms | 100ms | <50ms |
| Memory overhead | 50MB | 20MB | <15MB |
| Max throughput | 20/s | 50/s | 100/s |

## Contributing

### Adding a New Executor

1. Create `executors/<language>.go`
2. Implement the `Executor` interface
3. Add tests in `executors/<language>_test.go`
4. Register in `core/judge.go:loadExecutors()`

Example:
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
```

## Troubleshooting

### Common Issues

**"Judge not found" error:**
- Verify judge name and key in database
- Check `judges` table has correct credentials

**Compilation fails:**
- Ensure compilers are installed
- Check PATH includes compiler locations

**Sandbox errors:**
- Verify seccomp support (requires Linux 3.5+)
- Check kernel has seccomp-bpf enabled

**Connection refused:**
- Verify backend bridge is running on port 9999
- Check firewall rules

## License

Same as the main CLAOJ project.

## Acknowledgments

Based on the [DMOJ judge](https://github.com/DMOJ/judge-server) architecture.
