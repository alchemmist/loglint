# loglint

`loglint` is a Go static analyzer that checks log messages for common quality and safety issues. It is designed to run as a `golangci-lint` plugin and focuses on message text hygiene and sensitive data leaks.

## Rules

| # | Rule | Description |
|---|------|-------------|
| 1 | **lowercase_start** | Log messages must start with a lowercase letter |
| 2 | **english_only** | Log messages must contain English text only |
| 3 | **no_special_chars** | Log messages must not contain emojis, special characters, or repeated punctuation |
| 4 | **no_sensitive_data** | Log messages must not include sensitive data (by keyword matching) |

## Supported loggers

- `log/slog` (standard structured logger)
- `go.uber.org/zap` (Logger and SugaredLogger)
- `log` (standard library logger)

## Installation

### Build from source

```bash
make build
```

### Build the golangci-lint plugin

```bash
make plugin
```

### Standalone CLI

The `cmd/loglint` entrypoint is currently a stub and does not yet run the analyzer. Use the `golangci-lint` plugin integration for now.

## Usage

### As a golangci-lint plugin

1. Build the plugin:

```bash
make plugin
```

2. Add it to `.golangci.yml`:

```yaml
linters-settings:
  custom:
    loglint:
      path: ./loglint.so
      description: "Linter for checking log messages"
      original-url: github.com/alchemmist/loglint

linters:
  enable:
    - loglint
```

3. Run:

```bash
golangci-lint run
```

## Configuration

Create a configuration file in your project root:

- `.loglint.yml`
- `.loglint.yaml`
- `.loglint.json`

The analyzer searches the current directory and all parent directories.

### Example (YAML)

```yaml
rules:
  lowercase_start: true
  english_only: true
  no_special_chars: true
  no_sensitive_data: true

patterns:
  sensitive_keywords:
    - password
    - secret
    - token
    - api_key
    - private_key
    - my_custom_secret
```

### Example (JSON)

```json
{
  "rules": {
    "lowercase_start": true,
    "english_only": true,
    "no_special_chars": true,
    "no_sensitive_data": true
  },
  "patterns": {
    "sensitive_keywords": ["password", "secret", "token", "api_key", "private_key"]
  }
}
```

## Auto-fix

The analyzer can apply suggested fixes for:

- Rule 1: convert the first character to lowercase
- Rule 3: remove emojis/special characters and collapse repeated punctuation

Use the `-fix` flag when running through `golangci-lint`.

## Notes and limitations

- Rules 1-3 only run when the log message is a string literal.
- Rule 4 scans identifiers inside the message expression and subsequent arguments for sensitive keywords.

## Development

```bash
# Run tests
make test

# Run tests with coverage report
make test-cover

# Run vet, formatting checks, golangci-lint, and staticcheck
make check

# Format the codebase (gofmt + gofumpt)
make fmt
```

## Project layout

```
cmd/           # Standalone CLI entrypoint (stub)
pkg/analyzer/  # Analyzer implementation, rules, and config
plugin/        # golangci-lint plugin
testdata/      # Analyzer test fixtures
```

## License

MIT

## References

- https://disaev.me/p/writing-useful-go-analysis-linter/
- https://arslan.io/2019/06/13/using-go-analysis-to-write-a-custom-linter/
