# Contributing to StateLedger

Thank you for your interest in contributing to StateLedger!

## Development Setup

1. **Prerequisites**
   - Go 1.21 or later
   - Git
   - SQLite (included via modernc.org/sqlite)

2. **Clone and Build**
   ```bash
   git clone https://github.com/Retr0-XD/StateLedger.git
   cd StateLedger
   go build -o stateledger ./cmd/stateledger
   ```

3. **Run Tests**
   ```bash
   go test ./...
   ```

## Code Organization

```
cmd/stateledger/     # CLI application entry point
internal/
  ledger/            # Core append-only ledger
  collectors/        # Payload schemas and validation
  manifest/          # Manifest format
  sources/           # Real collectors (Git/Env/Config)
  artifacts/         # Content-addressable storage
```

## Testing Requirements

- All new features must include unit tests
- Integration tests for CLI commands
- Maintain or improve code coverage
- All tests must pass before PR merge

## Code Style

- Follow standard Go conventions (gofmt, golint)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and small

## Pull Request Process

1. **Fork the repository**
2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write tests first (TDD encouraged)
   - Implement feature
   - Run tests: `go test ./...`
   - Format code: `go fmt ./...`

4. **Commit with clear messages**
   ```bash
   git commit -m "Add feature: description"
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **PR Requirements**
   - Clear description of changes
   - Tests pass in CI
   - Code coverage maintained
   - No merge conflicts

## Adding New Collectors

1. Define payload schema in `internal/collectors/`
2. Add validation method with tests
3. Implement capture logic in `internal/sources/`
4. Update manifest dispatcher
5. Add CLI command if needed
6. Update documentation

## Reporting Issues

- Use GitHub Issues
- Include reproduction steps
- Provide relevant logs/output
- Specify Go version and OS

## Questions?

Open a GitHub Discussion or issue for questions about:
- Architecture decisions
- Feature proposals
- Implementation approaches

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
