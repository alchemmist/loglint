# loglint

`loglint` is a Go static analyzer for checking log messages. It works as a standalone tool or a `golangci-lint` plugin and supports the standard `log` package, `log/slog`, and `go.uber.org/zap`.

It checks that log messages:

- start with a lowercase letter, use English characters, contain no special characters, and do not expose sensitive data.

Build it with `make build`, then run `./loglint ./...`. Use `make plugin` to build the `golangci-lint` plugin and `make test` to run the tests; rules and sensitive keywords can be configured in `.loglint.yml`, `.loglint.yaml`, or `.loglint.json`.
