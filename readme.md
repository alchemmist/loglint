# loglint

![go version](https://img.shields.io/badge/go-1.25.0-00ADD8?logo=go&logoColor=white)
![license](https://img.shields.io/badge/license-MIT-2ea44f)
![type](https://img.shields.io/badge/type-go%2Fanalysis-555555)

`loglint` is a Go static analyzer that checks log messages for quality and safety issues. It runs as a `golangci-lint` plugin or as a standalone analyzer (singlechecker).

## What it checks

| # | Rule | Description |
|---|------|-------------|
| 1 | **lowercase_start** | Log messages must start with a lowercase letter (string literals only) |
| 2 | **english_only** | Log messages must not contain non-ASCII letters (string literals only; emojis ignored) |
| 3 | **no_special_chars** | Log messages must not contain emojis or non-alnum characters (string literals only) |
| 4 | **no_sensitive_data** | Looks for sensitive keywords in identifiers in the message expression and in identifiers or string literals in arguments after the message |

## Supported loggers

- `log/slog` with `Info/Warn/Error/Debug`, `*Context`, and `Log`
- `go.uber.org/zap` with `SugaredLogger` (`Info/Infof/Infow/...`) and `Logger` (`Info/Warn/Error/...`)
- `log` (standard library) with `Print*`, `Fatal*`, and `Panic*`

The analyzer also recognizes common receiver names like `log`, `logger`, `l`, `s`, `sugar`, and `zap` when deciding how to extract the message argument.

## Installation

### Build from source

```bash
make build
```

### Build the golangci-lint plugin

```bash
make plugin
```

## Usage

### Standalone CLI

```bash
./loglint ./...
```

Flags:

- `-config /path/to/.loglint.yml`
- `-fix` (apply suggested fixes)

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

If your `golangci-lint` version supports analyzer fixes, run it with `--fix` to apply `loglint` suggestions.

## Configuration

Supported file names:

- `.loglint.yml`
- `.loglint.yaml`
- `.loglint.json`

The analyzer searches the current directory and all parent directories. You can also pass `-config` to point to a specific file.

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

If `sensitive_keywords` is empty, the default keyword list is used.

## Auto-fix

Suggested fixes are provided for:

- **lowercase_start**: lowercases the first character
- **no_special_chars**: removes emojis and non-alnum characters (collapses extra spaces)

Apply fixes via `-fix` in the standalone CLI or via `golangci-lint --fix` if supported.

## Notes and limitations

- Rules 1-3 only run when the log message is a string literal.
- The sensitive data rule does not scan the literal message text itself. It scans identifiers in the message expression and identifiers or string literals in arguments after the message.

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

# Clean build artifacts and local caches
make clean
```

## Project layout

```
cmd/           # Standalone CLI entrypoint
pkg/analyzer/  # Analyzer implementation, rules, and config
plugin/        # golangci-lint plugin
testdata/      # Analyzer test fixtures
```

## License

MIT

## References

- https://disaev.me/p/writing-useful-go-analysis-linter/
- https://arslan.io/2019/06/13/using-go-analysis-to-write-a-custom-linter/
