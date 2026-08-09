# Contributing to Nginx Formatter

Thanks for taking the time to contribute! This project is a small, fast nginx configuration formatter written in Go. Contributions of all kinds are welcome: bug reports, feature requests, documentation improvements, and code.

<p style="text-align: center;">
  <a href="CONTRIBUTING.md" target="_blank">ENGLISH</a> | <a href="CONTRIBUTING_CN.md">中文</a>
</p>

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold it. Please report unacceptable behavior by opening an issue.

## Ways to Contribute

- **Report bugs**: Open an [issue](https://github.com/soulteary/nginx-formatter/issues) with a clear description and, ideally, a minimal nginx config that reproduces the problem (input + expected output vs. actual output).
- **Request features**: Open an issue describing the use case and why it would help.
- **Improve docs**: Fix typos, clarify wording, or translate. Both `README.md` and `README_CN.md` should stay in sync.
- **Submit code**: Fix a bug or implement a feature via a pull request (see below).

## Development Setup

### Prerequisites

- [Go 1.26+](https://go.dev/dl/) (the required version is pinned in [`go.mod`](go.mod))
- Git
- Optional: Docker, if you want to test the container images

### Getting Started

```bash
# Clone your fork
git clone https://github.com/<your-username>/nginx-formatter.git
cd nginx-formatter

# Download dependencies
go mod download

# Build the binary
go build -o nginx-formatter .

# Run it
./nginx-formatter -input=./your-dir-path
```

### Running the WebUI locally

```bash
go run . -web -port=8080
```

Then open http://localhost:8080 in your browser.

## Project Structure

```
main.go               Program entry point (argument parsing + dispatch)
internal/
  checker/            Runtime environment / input validation
  cmd/                Command-line argument parsing
  define/             Shared constants and defaults
  formatter/          High-level formatting entry point
  nginx/              Native Go AST-based nginx parser
    lexer.go          Tokenizer
    parser.go         Parser (tokens -> AST)
    ast.go            AST node definitions
    printer.go        AST -> formatted output
    token.go          Token definitions
  server/             Fiber-based WebUI (+ embedded assets)
  updater/            Reads/writes config files in a directory
  version/            Version string
docker/               Dockerfiles
.github/workflows/    CI: coverage, security scan, release
```

Most formatting logic lives in `internal/nginx/`. If you are fixing a formatting bug, that is usually the place to look.

## Testing

Run the full test suite before submitting a pull request:

```bash
go test ./...
```

Run tests with coverage (this mirrors what CI does):

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out
```

If you fix a bug or add a feature, please add a test case that covers it. The parser and formatter tests in `internal/nginx/nginx_test.go` and `internal/formatter/formatter_test.go` are good examples of the table-driven style used in this project.

## Code Style

- Format all code with `gofmt` before committing:

```bash
gofmt -w .
```

- Run `go vet ./...` to catch common mistakes.
- Follow standard Go conventions and keep changes focused and minimal.
- Only add comments that explain non-obvious intent; avoid comments that merely restate the code.

## Pull Request Process

1. Fork the repository and create a topic branch from `main`:

```bash
git checkout -b fix/some-bug
```

2. Make your changes, add tests, and make sure everything passes:

```bash
gofmt -w .
go vet ./...
go test ./...
```

3. Commit with a clear, descriptive message. Keep commits logically scoped.
4. Push your branch and open a pull request against `main`.
5. In the PR description, explain **what** changed and **why**. Link any related issues (e.g. `Closes #123`).
6. Make sure CI passes. A maintainer will review your PR and may request changes.

## Reporting Security Issues

For security-related reports, please follow the process described in [`SECURITY.md`](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the same license as the project ([Apache-2.0](LICENSE)).
