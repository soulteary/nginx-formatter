# Nginx Formatter

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) [![Go Report Card](./.github/goreportcard.svg)](./.github/goreportcard-report.md) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

<p style="text-align: center;">
  <a href="README.md" target="_blank">ENGLISH</a> | <a href="README_CN.md">中文文档</a>
</p>

<img src=".github/logo.png" width="120" >

Nginx configuration formatter ~10MB size, support CLI, WebUI, x86, ARM, Linux, macOS.

<img src=".github/preview.png">

> **What's new in [v2.6.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.6.0)**
>
> - Embedded scripts are now opaque to the printer's text passes. Everything between Lua long brackets (`[[ ]]`, `[=[ ]=]`, `--[[ ]]`) is string content, but the whole-document passes could not tell nginx structure from embedded script: blank runs inside a long string were collapsed, trailing spaces were trimmed, the body was re-indented, and a blank line was injected after every line starting with `}` — which is how a Lua table closes. `*_by_lua_block` bodies now round-trip byte for byte, while ordinary Lua outside a bracket is still normalized to the block indent.
> - CRLF files keep their line endings. The printer joins structural lines with `\n`, but a `\r` inside a comment, a raw block body or a multi-line quoted string is part of that token's text, so the result came back with *mixed* endings. Input is normalized to LF for parsing and converted back afterwards.
> - A leading UTF-8 BOM is removed. Windows editors add one silently and nginx has no idea what it is — the first directive's name became `<U+FEFF>server` rather than `server`, so nginx refused the file with `unknown directive` while the formatter reported success. The removal is reported per file, and only once it has actually happened: a file that fails to parse is left alone and nothing is claimed about it.
> - `sites-available/default` and `sites-enabled/default` are formatted. The scan set was `*.conf` only, so on Debian and Ubuntu the most common site file there is — and the only one with no extension — was skipped without a word. A file named `default` anywhere else in the tree is still left alone.
> - `nginx-formatter format /etc/nginx` now formats the path you named. The positional argument was parsed and silently discarded, so it walked away and reformatted the *working directory* instead, exit code 0.
> - **Behavior change:** the container-root guard exits non-zero. `format` inside Docker with the working directory at `/` printed its hint and exited **0**, so a CI job saw a clean run when nothing had been formatted — usually caused by a forgotten `-v` mount, which is exactly the case a pipeline should catch.
> - **Behavior change:** `serve --port 65535` is accepted. The guard read `port >= 65535` while its message promised everything "within 65535"; the accepted range is now stated as `1025..65535` and enforced as written.
> - An output directory nested inside the input tree is excluded from the scan, instead of being re-ingested on the next run and nesting one level deeper each time (`out/`, `out/out/`, ...).
> - Long filenames format again, fixing a v2.5.0 regression: the temporary file used for the atomic write is 16-22 bytes longer than its target, so a legal `.conf` name could push it past `NAME_MAX` and the file could not be formatted at all.
> - New: `--check` lists the files that are not formatted and exits 1 if any are (the `gofmt -l` shape), and `--diff` prints a unified diff of what would change. Neither writes anything, and neither creates the output directory.
> - New: `--input -` reads stdin and writes stdout, for an editor's format-on-save hook.
> - New: `--quiet` / `-q` keeps the banner and progress narration off stdout. Errors are unaffected — they still go to stderr and still exit non-zero.
> - stdout is now usable by a program. `--check`, `--diff` and stdin own it, so the banner, the progress narration, the scan's skip notices and the per-file failure lines all go to stderr or are suppressed there. `--quiet` never swallows a *result*: `--check -q` still prints the file list.

> **What's new in [v2.5.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.5.0)**
>
> - Fixed several cases where formatting silently corrupted the configuration it overwrote. A quote inside a bare word (`sub_filter href="/old" href="/new";`, `alias /data/o'brien/;`) was treated as a token boundary and the pieces were rejoined with spaces, changing the directive's argument count so nginx refused to load the file. Bare-word termination now matches nginx's own tokenizer.
> - Removed the regex-based `return` normalization that ran over the whole file before parsing. It re-quoted single-quoted arguments — `return 200 'ok';` became `return 200 "'ok'";`, changing the body nginx serves — and rewrote `return` text inside `*_by_lua_block` bodies, comments and quoted strings. The AST parser has made it unnecessary since v2.0.
> - A comment on an empty block's closing brace (`upstream backend {` / `} # TODO`) is no longer deleted, and formatting is now idempotent: running the formatter twice produces the same file.
> - Input containing a NUL byte or invalid UTF-8 is now rejected instead of being silently truncated or rewritten with replacement characters, and the input file is left untouched. An unterminated quote, `${`, or trailing backslash is reported as an error rather than repaired with an extra `;` on every run.
> - **Behavior change:** `format -i <dir>` without `--output` now formats that directory in place, matching single-file mode. It previously wrote the formatted tree into the current working directory, which left the target untouched, could overwrite same-named files in the working directory, and made both documented Docker commands no-ops.
> - **Behavior change:** formatted output now ends with exactly one newline. Configurations ending in a directive previously lost their trailing newline, so expect a one-time diff on the first run.
> - Fixed the documented indent characters: `--char '\s'` wrote literal backslash-s into every line while reporting `[SPACE]`, and `--char '\t'` was rejected outright. Both now resolve to a real space and tab.
> - Symbolic links are reported and skipped rather than followed, so the standard `sites-enabled` → `sites-available` layout is formatted once through the real file instead of aborting the entire run.
> - Files are written atomically (temporary file plus rename) and keep their existing permissions and ownership, instead of being truncated in place and forced to mode 0600. One unparseable file no longer stops the rest of the batch.
> - Added `serve --host` to restrict the WebUI listener, plus a 1 MiB request body limit and a 64-level nesting cap — indentation grows with the square of the nesting depth, so a few kilobytes of nested blocks could previously exhaust memory. Formatting errors now show the parser's message and line number instead of a bare `format error`.
>
> **What's new in [v2.4.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.4.0)**
>
> - Fixed the WebUI silently deleting nginx variables. The formatted output was spliced into the page with `regexp.ReplaceAllString`, where `$name` is a capture-group reference, so `$host`, `$remote_addr` and `$upstream_addr` were replaced with empty strings.
> - The WebUI now HTML-escapes the formatted output, so a configuration containing `</textarea>` or `<` can no longer break the page or inject markup.
> - The result is now returned directly from `POST /format` instead of being parked in a process-wide cache and picked up by a redirect, so one visitor's configuration is never served to another. `GET /` always returns the default page.
> - Pinned the `gosec` security scan to a release commit SHA instead of tracking the `master` branch.
>
> **What's new in [v2.3.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.3.0)**
>
> - Updated the automated Go Report Card workflow to use `soulteary/goreportcard-action` v1.1.0.
> - Hardened the Codecov workflow by restricting `GITHUB_TOKEN` to read-only repository contents.
> - Published checksums and prebuilt binaries for macOS and Linux on x86, x86-64, ARM64, ARMv6, and ARMv7.
>
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
docker pull soulteary/nginx-formatter:v2.5.0
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

### GitHub Actions

To check Nginx configuration formatting automatically in pull requests, use [soulteary/nginx-format-action](https://github.com/soulteary/nginx-format-action). Create `.github/workflows/nginx-format.yml`:

```yaml
name: Nginx format

on:
  pull_request:
    paths:
      - "**/*.conf"

permissions:
  contents: read

jobs:
  format:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: soulteary/nginx-format-action@v1
        with:
          path: .
          mode: check
```

See the [Nginx Format Action documentation](https://github.com/soulteary/nginx-format-action#readme) for write mode, indentation settings, version pinning, and more examples.

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

Format the configuration file in the specified directory. Without `--output` the
files are formatted in place, matching single-file behaviour:

```bash
./nginx-formatter format -i ./your-dir-path
```

Symbolic links are reported and skipped rather than followed, so the usual
`sites-enabled` → `sites-available` layout is formatted exactly once, through
the real file, and the links are left intact.

The scanned set is every `*.conf` file, plus `default` when it sits directly in
a `sites-enabled` or `sites-available` directory — on Debian and Ubuntu that is
the site file the nginx package ships, and the only one with no extension. A
file named `default` anywhere else is left alone.

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

The WebUI binds every interface by default so the Docker usage below works.
Restrict it to this machine with `--host`:

```bash
./nginx-formatter serve --host 127.0.0.1
```

### CI: check without writing

`--check` reports which files are not formatted and exits 1 if any are; `--diff`
prints a unified diff of what would change. Neither writes anything, and their
stdout carries only that output, so it can be piped:

```bash
# Fail the build if anything is unformatted
./nginx-formatter format -i ./conf.d --check

# See exactly what would change
./nginx-formatter format -i ./conf.d --diff
```

### stdin / stdout

`--input -` reads the configuration from stdin and writes the result to stdout,
which is what an editor's format-on-save hook expects:

```bash
cat nginx.conf | ./nginx-formatter format -i -
```

### Quiet output

`--quiet` / `-q` keeps the banner and the per-file progress narration off
stdout. It is a persistent flag, so it works on every subcommand:

```bash
./nginx-formatter format -i ./conf.d --quiet
```

Errors are deliberately not suppressed. They go to stderr and still exit
non-zero, so a pipeline keeps a clean stdout without losing the reason a run
failed:

```bash
$ ./nginx-formatter format -i broken.conf -q
2026/09/18 12:05:13 line 2: unexpected EOF, missing '}'
$ echo $?
1
```

`--quiet` never swallows a **result**. `--check` still prints its file list,
`--diff` still prints its patch, and `-i -` still prints the formatted
configuration — the flag removes narration, not the output you asked for.

### Byte order marks

Windows editors add a UTF-8 BOM silently, and nginx has no idea what it is:
with one present the first directive's name is `<U+FEFF>server` rather than
`server`, and nginx refuses the file with `unknown directive`. The formatter
removes it and says so, per file:

```
Formatter Nginx Conf bom.conf had a UTF-8 BOM; removed it (nginx rejects a config that starts with one)
```

Because that edits the file rather than only its formatting, the notice appears
only once the removal has actually happened. A file that fails to parse is left
exactly as it was, and with `--output` pointing elsewhere the notice names the
written copy and says the input is unchanged. A BOM sequence anywhere other
than the very start is ordinary content and is left alone.

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
  -q, --quiet     Suppress the banner and progress output (errors still go to stderr)
  -v, --version   version for nginx-formatter
```

`format` flags:

```bash
  -c, --char \s         Indent char (space/tab/\s/`\t`) (default " ")
      --check           Do not write; list files that are not formatted and exit 1 if any
      --diff            Do not write; print a unified diff of what would change and exit 1 if any
  -h, --help            help for format
  -n, --indent int      Indent size (default 2)
  -i, --input string    Input directory or file, or "-" for stdin (default: current directory)
  -o, --output string   Output directory or file path

Global Flags:
  -q, --quiet   Suppress the banner and progress output (errors still go to stderr)
```

`serve` flags:

```bash
  -c, --char \s       Default indent char the WebUI applies (space/tab/\s/`\t`) (default " ")
  -h, --help          help for serve
      --host string   Address to bind (default: all interfaces)
  -n, --indent int    Default indent size the WebUI applies (default 2)
  -p, --port int      WebUI port (default 8080)

Global Flags:
  -q, --quiet   Suppress the banner and progress output (errors still go to stderr)
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
