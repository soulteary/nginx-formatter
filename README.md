# Nginx Formatter

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) ![Go Report Card](./.github/goreportcard.svg) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

<p style="text-align: center;">
  <a href="README.md" target="_blank">ENGLISH</a> | <a href="README_CN.md">中文文档</a>
</p>

<img src=".github/logo.png" width="120" >

Nginx configuration formatter ~10MB size, support CLI, WebUI, x86, ARM, Linux, macOS.

<img src=".github/preview.png">

> **What's new in v2.2.0**
>
> - Redesigned the CLI with semantic subcommands (`format` / `serve` / `version`) built on [Cobra](https://github.com/spf13/cobra), with modern `--long`/`-short` flags and per-command `--help`.
> - Full backward compatibility: the legacy single-dash long flags (`-input`, `-output`, `-indent`, `-char`, `-web`, `-port`) still work, so existing scripts and Docker commands keep running unchanged.
>
> **What's new in v2.1.0**
>
> - Support formatting a single file: when `-input` points to a file, only that file is formatted (any extension is accepted, not just `.conf`), and `-output` can overwrite in place, target a directory, or write to a specific file path.
> - Added [Homebrew](https://github.com/soulteary/homebrew-tap) installation support for macOS / Linux.
> - Added a [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).
>
> **What's new in v2.0.0**
>
> - Rewrote the formatting engine with a native Go AST-based nginx parser, replacing the previous `goja` / `beautifier.js` runtime. No JavaScript runtime dependency anymore, faster and lighter.
> - Switched the WebUI to the [Fiber](https://github.com/gofiber/fiber) web framework.
> - Upgraded to Go 1.26 with root-scoped filesystem access for safer directory traversal.
> - Various bug fixes: preserve backslash escapes in bare words, tolerate directives missing a trailing semicolon before `}` or EOF, and smarter `return` directive normalization.

## Download

Download the binaries for your system and architecture from the [releases page](https://github.com/soulteary/nginx-formatter/releases).

<img src=".github/dockerhub.png" width="80%" >

If you use docker, you can use the following command ([DockerHub](https://hub.docker.com/r/soulteary/nginx-formatter)):

```bash
docker pull soulteary/nginx-formatter:latest
docker pull soulteary/nginx-formatter:v2.2.0
```

### Homebrew

On macOS / Linux you can install it via [Homebrew](https://github.com/soulteary/homebrew-tap):

```bash
brew tap soulteary/tap
brew install soulteary/tap/nginx-formatter
```

After installation the `nginx-formatter` command is available globally, so you can run it directly (without the `./` prefix used below):

```bash
nginx-formatter serve
```

To upgrade or uninstall later:

```bash
brew upgrade soulteary/tap/nginx-formatter
brew uninstall nginx-formatter
```

## Usage

Since v2.2.0 the CLI uses semantic subcommands (`format` / `serve` / `version`) with modern `--long`/`-short` flags. The old single-dash long flags (`-input`, `-output`, `-indent`, `-char`, `-web`, `-port`) remain fully supported for backward compatibility, so existing scripts and Docker commands keep working.

Use default parameters to format all configuration files in the current directory:

```bash
./nginx-formatter format
```

### Common Usage (CLI & WebUI)

Use different indentation symbols (you can use spaces, tabs, `space`, `tab`, `\s`, `\t`) and indentation amounts:

```bash
./nginx-formatter format -n 4 -c space
```

### CLI Usage

Format the configuration file in the specified directory:

```bash
./nginx-formatter format -i ./your-dir-path
```

Format a directory and save it in a new directory:

```bash
./nginx-formatter format -i ./your-dir-path -o ./your-output-dir
```

Format a single file: when `--input` points to a file, only that file is formatted (any file extension is accepted, not just `.conf`). The `--output` value has three meanings in single-file mode:

- empty: overwrite the input file in place
- an existing directory: write to `<output-dir>/<original-file-name>`
- otherwise: treat it as a target file path, creating the parent directory if needed

```bash
# overwrite in place
./nginx-formatter format -i ./nginx.conf

# write into an existing directory
./nginx-formatter format -i ./nginx.conf -o ./dist

# write to a specific file path
./nginx-formatter format -i ./nginx.conf -o ./dist/nginx.formatted.conf
```

### WebUI Usage

Start the web interface:

```bash
./nginx-formatter serve
```

specified the port:

```bash
./nginx-formatter serve -p 8123
```

`--indent` and `--char` set the default indentation the WebUI applies when formatting:

```bash
./nginx-formatter serve -p 8123 -n 4 -c space
```

### Version

Print the version:

```bash
./nginx-formatter version
```

### Backward compatibility (legacy flags)

Old-style single-dash long flags still work exactly as before:

```bash
# Format a directory (legacy)
./nginx-formatter -input=./your-dir-path -output=./your-output-dir -indent=4 -char=" "

# Format a single file (legacy)
./nginx-formatter -input=./nginx.conf

# Start the WebUI (legacy)
./nginx-formatter -web -port=8123
```

### Docker Usage

There is no difference between using parameters in Docker and the above, for example, we start a Web UI formatting tool service in Docker:

```bash
# new subcommand style
docker run --rm -it -p 8080:8080 soulteary/nginx-formatter:latest serve

# legacy style still works
docker run --rm -it -p 8080:8080 soulteary/nginx-formatter:latest -web
```

If you want to format the configuration of the current directory, you can use the program in Docker with a command similar to the following:

```bash
# new subcommand style
docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter:latest format -i /app

# legacy style still works
docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter:latest -input=/app
```

## Full parameters supported

Run `nginx-formatter --help` to see the available commands, or `nginx-formatter <command> --help` for a command's flags and examples:

```bash
Usage:
  nginx-formatter [flags]
  nginx-formatter [command]

Available Commands:
  format      Format Nginx configuration files in a directory or a single file
  serve       Start the browser-based WebUI
  version     Print the version number

Flags:
  -h, --help      help for nginx-formatter
  -v, --version   version for nginx-formatter
```

`format` flags:

```bash
  -c, --char string     Indent char (space/tab/\s/\t) (default " ")
  -n, --indent int      Indent size (default 2)
  -i, --input string    Input directory or file (default: current directory)
  -o, --output string   Output directory or file path
```

`serve` flags:

```bash
  -c, --char string   Default indent char the WebUI applies (space/tab/\s/\t) (default " ")
  -n, --indent int    Default indent size the WebUI applies (default 2)
  -p, --port int      WebUI port (default 8080)
```

## Contributing

Contributions are welcome! Please read the [Contributing Guide](CONTRIBUTING.md) to get started, and follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Credits

Formatter Components

- Slomkowski Created a beautifier for nginx config files with Python under [Apache-2.0 license], 24/06/2016
  - https://github.com/1connect/nginx-config-formatter (https://github.com/slomkowski/nginx-config-formatter)
- Yosef Ported the JavaScript beautifier under [Apache-2.0 license], 24/08/2016
  - https://github.com/vasilevich/nginxbeautifier
- soulteary Modify the JavaScript version for golang execution, under [Apache-2.0 license], 18/04/2023:
  - simplify the program, fix bugs, improve running speed, and allow running in golang
  - https://github.com/soulteary/nginx-formatter
- soulteary Rewrote the formatter with a native Go AST-based nginx parser (dropping the JavaScript runtime), under [Apache-2.0 license], since v2.0.0:
  - https://github.com/soulteary/nginx-formatter

Web Components

- Fiber is an Express inspired web framework written in Go (Golang), under [MIT license].
  - https://github.com/gofiber/fiber
- Code Mirror, in-browser code editor, under [MIT license].
  - https://github.com/codemirror/codemirror5
