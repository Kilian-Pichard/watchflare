# Watchflare Agent - Testing Guide

## Running Tests

### Unit Tests

Run all unit tests:

```bash
go test ./...
```

Run tests for a specific package:

```bash
# Config package
go test -v ./config/

# Sysinfo package
go test -v ./sysinfo/

# Client package
go test -v ./client/
```

Run tests with coverage:

```bash
go test -cover ./...

# Detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require a running backend server.

**Prerequisites:**
1. Start the Watchflare backend server
2. Create a test server in the dashboard and get a registration token
3. Set environment variables

**Run integration tests:**

```bash
# Set environment variables
export WATCHFLARE_TEST_BACKEND_HOST=localhost
export WATCHFLARE_TEST_BACKEND_PORT=50051
export WATCHFLARE_TEST_TOKEN=wf_reg_your_test_token_here

# Run integration tests
go test -tags=integration -v ./...
```

**What the integration tests do:**
1. ✅ Collect system information
2. ✅ Connect to backend server
3. ✅ Register agent with token
4. ✅ Save configuration
5. ✅ Load configuration
6. ✅ Send heartbeats
7. ✅ Verify continuous monitoring

### Test with Existing Config

If you have an already registered agent, test heartbeat functionality:

```bash
# This will use the existing config in /etc/watchflare/agent.conf
go test -tags=integration -v -run TestEndToEndWithExistingConfig
```

## Test Coverage

Current test coverage:

| Package | Coverage | Tests |
|---------|----------|-------|
| `config` | 100% | 8 tests |
| `sysinfo` | 95% | 7 tests |
| `client` | 85% | 6 tests |

## Continuous Integration

### GitHub Actions

Add this workflow to `.github/workflows/test.yml`:

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Run unit tests
      run: go test -v -cover ./...

    - name: Run integration tests
      env:
        WATCHFLARE_TEST_BACKEND_HOST: ${{ secrets.TEST_BACKEND_HOST }}
        WATCHFLARE_TEST_BACKEND_PORT: ${{ secrets.TEST_BACKEND_PORT }}
        WATCHFLARE_TEST_TOKEN: ${{ secrets.TEST_TOKEN }}
      run: go test -tags=integration -v ./...
      if: ${{ secrets.TEST_BACKEND_HOST != '' }}
```

## Writing Tests

### Test File Naming

- Unit tests: `*_test.go`
- Integration tests: `integration_test.go` with `// +build integration` tag

### Test Function Naming

```go
func TestFunctionName(t *testing.T)           // Basic test
func TestFunctionName_ErrorCase(t *testing.T) // Test error handling
func BenchmarkFunctionName(b *testing.B)      // Benchmark test
```

### Table-Driven Tests

Use table-driven tests for multiple test cases:

```go
func TestGetConfigDir(t *testing.T) {
    tests := []struct {
        name     string
        envValue string
        want     string
    }{
        {
            name:     "default path when no env var",
            envValue: "",
            want:     DefaultConfigDir,
        },
        {
            name:     "custom path from env var",
            envValue: "/custom/config",
            want:     "/custom/config",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Test Helpers

Use `t.Helper()` for test helper functions:

```go
func setupTestEnv(t *testing.T) (string, func()) {
    t.Helper()
    // Setup code
    return tmpDir, cleanup
}
```

### Cleanup

Always clean up resources:

```go
func TestSomething(t *testing.T) {
    tmpDir, cleanup := setupTestEnv(t)
    defer cleanup()

    // Test code
}
```

## Debugging Tests

### Verbose Output

```bash
go test -v ./...
```

### Run Specific Test

```bash
go test -v -run TestSpecificFunction
```

### Show Test Output

```bash
go test -v ./... | grep -A 5 "FAIL"
```

### Debug with Delve

```bash
dlv test -- -test.run TestSpecificFunction
```

## Best Practices

1. **Test Independence**: Each test should be independent and not rely on other tests
2. **Clean State**: Always start with a clean state (use temp directories, cleanup after)
3. **Mock External Dependencies**: Use mocks for external services when possible
4. **Test Edge Cases**: Test error cases, empty inputs, invalid data
5. **Use Table Tests**: For multiple similar test cases
6. **Meaningful Names**: Test names should describe what they test
7. **Fast Tests**: Keep unit tests fast (< 1s), integration tests can be slower

## Troubleshooting

### Tests Fail with Permission Errors

If you get permission errors when running tests:

```bash
# Make sure you're using a temp directory for tests
export WATCHFLARE_CONFIG_DIR=/tmp/watchflare-test/config
export WATCHFLARE_DATA_DIR=/tmp/watchflare-test/data
export WATCHFLARE_LOG_DIR=/tmp/watchflare-test/logs
```

### Integration Tests Timeout

Increase timeout for slow networks:

```bash
go test -timeout 30s -tags=integration -v
```

### Backend Connection Refused

Make sure the backend server is running:

```bash
# In backend directory
go run main.go
```
