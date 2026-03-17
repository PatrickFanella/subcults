# Profiling Guide

This document describes how to use pprof profiling to identify performance bottlenecks in the Subcults API server.

## Overview

The API server includes Go's built-in pprof profiling capabilities for performance analysis. Profiling is **DISABLED BY DEFAULT** and should **NEVER** be enabled in production environments for security reasons.

## Security Warning

⚠️ **CRITICAL SECURITY NOTICE** ⚠️

Profiling endpoints expose sensitive information about your application:
- Memory contents (potentially including secrets like API keys, JWTs, passwords)
- Source code structure and file paths
- Resource usage patterns
- Performance characteristics
- Internal implementation details

**NEVER enable profiling in production environments.** Profiling is strictly for development and testing.

## Enabling Profiling

### Environment Variable

Set the `PROFILING_ENABLED` environment variable to enable profiling:

```bash
# Enable profiling (development only)
export PROFILING_ENABLED=true
export SUBCULT_ENV=development

# Start the API server
./bin/api
```

### Configuration File

You can also enable profiling in your YAML configuration file:

```yaml
# config.yaml
profiling_enabled: true
env: development
```

### Safety Checks

The middleware includes multiple safety checks:

1. Profiling is disabled by default
2. Environment must not be "production" or "prod"
3. Both checks must pass for profiling to be enabled
4. Warnings are logged when profiling is enabled

## Available Endpoints

When profiling is enabled, the following endpoints are available:

### Index Page
```
GET /debug/pprof/
```
HTML index page listing all available profiles.

### CPU Profile
```
GET /debug/pprof/profile?seconds=30
```
CPU profile for the specified duration (default: 30 seconds).

Example:
```bash
# Collect 30-second CPU profile
curl http://localhost:8080/debug/pprof/profile > cpu.prof

# Collect 60-second CPU profile
curl 'http://localhost:8080/debug/pprof/profile?seconds=60' > cpu.prof
```

### Memory Heap Profile
```
GET /debug/pprof/heap
```
Memory allocation profile (heap).

Example:
```bash
curl http://localhost:8080/debug/pprof/heap > heap.prof
```

### Goroutine Profile
```
GET /debug/pprof/goroutine
```
Stack traces of all current goroutines.

Example:
```bash
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
```

### Block Profile
```
GET /debug/pprof/block
```
Stack traces that led to blocking on synchronization primitives.

### Mutex Profile
```
GET /debug/pprof/mutex
```
Stack traces of holders of contended mutexes.

### Allocation Profile
```
GET /debug/pprof/allocs
```
All past memory allocations (sampling).

### Other Profiles
- `/debug/pprof/threadcreate` - Thread creation profile
- `/debug/pprof/cmdline` - Command line invocation
- `/debug/pprof/symbol` - Symbol lookup
- `/debug/pprof/trace` - Execution trace

### Profiling Status
```
GET /debug/profiling/status
```
Returns JSON indicating whether profiling is enabled. This endpoint is always available.

Example:
```bash
curl http://localhost:8080/debug/profiling/status
```

Response:
```json
{
  "profiling_enabled": true,
  "environment": "development",
  "status": "enabled",
  "endpoints": [
    "/debug/pprof/",
    "/debug/pprof/profile",
    "/debug/pprof/heap",
    ...
  ],
  "security_warning": "Profiling should NEVER be enabled in production"
}
```

## Using pprof Tool

### Interactive Mode

Use `go tool pprof` to analyze profiles interactively:

```bash
# CPU profile (waits 30 seconds while profiling)
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

# Heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

Once in interactive mode, use these commands:

```
# Show top functions by time/memory
(pprof) top

# Show top 20 functions
(pprof) top20

# Show cumulative top functions
(pprof) top -cum

# List source code for a function
(pprof) list <function_name>

# Show call graph
(pprof) web

# Generate flame graph (requires graphviz)
(pprof) web

# Exit
(pprof) quit
```

### Generate Flame Graphs

Flame graphs provide excellent visualization of performance hotspots:

```bash
# Generate CPU flame graph
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30

# Generate heap flame graph
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/heap
```

This opens a web UI at http://localhost:8081 with interactive flame graphs.

### Command Line Reports

Generate reports without interactive mode:

```bash
# Text report
go tool pprof -text http://localhost:8080/debug/pprof/profile?seconds=30

# PDF report (requires graphviz)
go tool pprof -pdf http://localhost:8080/debug/pprof/profile > profile.pdf

# PNG image (requires graphviz)
go tool pprof -png http://localhost:8080/debug/pprof/heap > heap.png
```

## Common Profiling Workflows

### Identifying CPU Bottlenecks

1. Run CPU profile during load:
   ```bash
   # In terminal 1: Start load test
   k6 run perf/k6/stream-load-test.js
   
   # In terminal 2: Collect CPU profile
   go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30
   ```

2. Look for:
   - Hot functions (top of flame graph)
   - Unexpected allocations
   - Time spent in locks or I/O

3. Focus optimization on the top 3-5 functions consuming CPU time

### Identifying Memory Leaks

1. Take heap snapshot after running for a while:
   ```bash
   go tool pprof -http=:8081 http://localhost:8080/debug/pprof/heap
   ```

2. Look for:
   - Large allocations in unexpected places
   - Growing memory usage over time
   - Objects that should be garbage collected

3. Use `inuse_space` (default) to see currently allocated memory
4. Use `-alloc_space` flag to see all allocations (including freed):
   ```bash
   go tool pprof -alloc_space -http=:8081 http://localhost:8080/debug/pprof/heap
   ```

### Identifying Goroutine Leaks

1. Check goroutine profile:
   ```bash
   go tool pprof -http=:8081 http://localhost:8080/debug/pprof/goroutine
   ```

2. Look for:
   - Unexpectedly high goroutine count
   - Goroutines stuck waiting on channels
   - Goroutines blocked on I/O or mutexes

3. Monitor goroutine count over time:
   ```bash
   # Check current goroutine count
   curl http://localhost:8080/debug/pprof/goroutine | head -1
   ```

### Investigating Lock Contention

1. Enable mutex profiling (requires code change in main.go):
   ```go
   import "runtime"
   
   func main() {
       runtime.SetMutexProfileFraction(1) // Enable mutex profiling
       // ... rest of main
   }
   ```

2. Collect mutex profile:
   ```bash
   go tool pprof -http=:8081 http://localhost:8080/debug/pprof/mutex
   ```

3. Look for:
   - High lock contention on specific mutexes
   - Locks held for long periods
   - Bottlenecks in concurrent code

## Benchmarking

Run Go benchmarks to measure performance before and after optimizations:

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkHTTPMetrics -benchmem ./internal/middleware

# Run benchmarks with profiling
go test -bench=BenchmarkHTTPMetrics -benchmem -cpuprofile=cpu.prof ./internal/middleware

# Analyze benchmark CPU profile
go tool pprof cpu.prof
```

Example benchmark output:
```
BenchmarkHTTPMetrics_Overhead/without_middleware-4    5000000    250 ns/op    0 B/op    0 allocs/op
BenchmarkHTTPMetrics_Overhead/with_middleware-4       3000000    400 ns/op   48 B/op    1 allocs/op
```

Interpret results:
- Iterations per second (higher is better)
- Nanoseconds per operation (lower is better)
- Bytes allocated per operation (lower is better)
- Allocations per operation (lower is better)

## Performance Budgets

Target performance for critical paths (from perf/):

- **API Latency**: p95 < 300ms
- **Stream Join**: < 2s
- **Map Render**: < 1.2s
- **Trust Recompute**: < 5m

Use profiling to ensure optimizations keep you within these budgets.

## Best Practices

1. **Profile in realistic conditions**
   - Use production-like data volumes
   - Simulate realistic load patterns
   - Profile during peak usage scenarios

2. **Profile before and after optimizations**
   - Establish baseline performance
   - Measure impact of changes
   - Avoid premature optimization

3. **Focus on the biggest wins**
   - Optimize hot paths first (top of flame graph)
   - 80/20 rule: 20% of code consumes 80% of resources
   - Don't optimize code that isn't a bottleneck

4. **Consider trade-offs**
   - CPU vs memory
   - Simplicity vs performance
   - Maintainability vs speed

5. **Document findings**
   - Record baseline metrics
   - Document optimization decisions
   - Track performance over time

## Troubleshooting

### Profiling endpoints return 404

Check that profiling is enabled:
```bash
curl http://localhost:8080/debug/profiling/status
```

If disabled, set `PROFILING_ENABLED=true` and restart the server.

### "SECURITY VIOLATION" in logs

Profiling was attempted in production environment. This is blocked for security.

Ensure `SUBCULT_ENV` is set to "development" or "dev".

### pprof fails to connect

Check that the API server is running and accessible:
```bash
curl http://localhost:8080/health/live
```

Verify network connectivity and firewall rules.

### No data in profile

The profile duration may be too short. Increase the `seconds` parameter:
```bash
go tool pprof 'http://localhost:8080/debug/pprof/profile?seconds=60'
```

Ensure the server is under load during profiling.

## Additional Resources

- [Go pprof documentation](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://go.dev/blog/pprof)
- [Go Performance Wiki](https://github.com/golang/go/wiki/Performance)
- [Flame Graphs](http://www.brendangregg.com/flamegraphs.html)

## See Also

- [Architecture Documentation](../README.md)
- [Performance Testing](../perf/k6/stream-load-test.js)
- [Observability](./OBSERVABILITY.md)
