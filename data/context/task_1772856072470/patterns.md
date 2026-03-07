# Project Patterns

## Go Project

### Code Style
- Use `gofmt` for formatting
- Error handling: return errors, don't panic
- Naming: CamelCase for exports, camelCase for internal

### Error Handling
- Return `(result, error)` tuples
- Wrap errors with context: `fmt.Errorf("doing X: %w", err)`
- Check errors immediately after function calls

### Testing
- Test files: `*_test.go` in same package
- Use `testing.T` for tests
- Table-driven tests preferred

