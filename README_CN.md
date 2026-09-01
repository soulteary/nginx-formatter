# Nginx Formatter / Nginx 格式化工具

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) ![Go Report Card](./.github/goreportcard.svg) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

<p style="text-align: center;">
  <a href="README.md" target="_blank">ENGLISH</a> | <a href="README_CN.md">中文文档</a>
</p>

<img src=".github/logo.png" width="120" >

一款 10MB 左右的，小巧、简洁的 Nginx 格式化工具，支持命令行、WebUI、Docker、x86、ARM、macOS、Linux。

<img src=".github/preview.png">

> **[v2.3.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.3.0) 更新说明**
>
> - 将自动化 Go Report Card 工作流升级至 `soulteary/goreportcard-action` v1.1.0。
> - 收紧 Codecov 工作流权限，将 `GITHUB_TOKEN` 明确限制为仅可读取仓库内容。
> - 发布适用于 macOS 与 Linux 的 x86、x86-64、ARM64、ARMv6、ARMv7 预编译程序及校验和。
>
> **v2.2.0 更新说明**
>
> - 基于 [Cobra](https://github.com/spf13/cobra) 重新设计命令行，改为语义化子命令（`format` / `serve` / `version`），采用现代的 `--long`/`-short` 参数风格，并为每个命令提供独立的 `--help`。
> - 完全向后兼容：旧版的单横线长参数（`-input`、`-output`、`-indent`、`-char`、`-web`、`-port`）依然可用，现有脚本与 Docker 命令无需改动即可继续运行。
>
> **v2.1.0 更新说明**
>
> - 支持格式化单个文件：当 `-input` 指向文件时，仅格式化该文件（不再限制 `.conf` 后缀，任意后缀均可），`-output` 可原地覆盖、写入目录或写入指定文件路径。
> - 新增 macOS / Linux 下的 [Homebrew](https://github.com/soulteary/homebrew-tap) 安装支持。
> - 新增[贡献指南](CONTRIBUTING_CN.md)与[行为准则](CODE_OF_CONDUCT.md)。
>
> **v2.0.0 更新说明**
>
> - 使用原生 Go AST 的 Nginx 解析器重写了格式化引擎，替换了此前基于 `goja` / `beautifier.js` 的运行时。不再依赖 JavaScript 运行时，运行更快、体积更小。
> - WebUI 切换到 [Fiber](https://github.com/gofiber/fiber) Web 框架。
> - 升级到 Go 1.26，并采用根目录限定（root-scoped）的文件系统访问，目录遍历更安全。
> - 多项问题修复：保留裸词中的反斜杠转义、容忍指令在 `}` 或文件结尾前缺失分号、更智能的 `return` 指令规范化处理。

## 程序下载

从[发布页面](https://github.com/soulteary/nginx-formatter/releases)下载适用于您系统和架构的二进制文件和压缩包。

<img src=".github/dockerhub.png" width="80%" >

如果使用 Docker，可以使用以下命令（[DockerHub](https://hub.docker.com/r/soulteary/nginx-formatter)）：

```bash
docker pull soulteary/nginx-formatter:latest
docker pull soulteary/nginx-formatter:v2.3.0
```

### Homebrew 安装

在 macOS / Linux 上，你可以通过 [Homebrew](https://github.com/soulteary/homebrew-tap) 安装：

```bash
brew tap soulteary/tap
brew install soulteary/tap/nginx-formatter
```

安装完成后，`nginx-formatter` 命令会全局可用，可以直接运行（无需下文中的 `./` 前缀）：

```bash
nginx-formatter serve
```

后续升级或卸载：

```bash
brew upgrade soulteary/tap/nginx-formatter
brew uninstall nginx-formatter
```

## 程序使用

从 v2.2.0 起，命令行改为语义化子命令（`format` / `serve` / `version`），并采用现代的 `--long`/`-short` 参数风格。旧版的单横线长参数（`-input`、`-output`、`-indent`、`-char`、`-web`、`-port`）依然完全兼容，因此现有脚本与 Docker 命令无需改动即可继续使用。

使用默认参数格式化当前目录中的所有的 Nginx 配置文件：

```bash
./nginx-formatter format
```

### 通用玩法 (CLI & WebUI)

使用不同的缩进符号（可以使用空格、制表符、`space`、`tab`、`\s`、`\t`）和缩进量：

```bash
./nginx-formatter format -n 4 -c space
```

### 命令行用法（CLI）

格式化指定目录中的配置文件：

```bash
./nginx-formatter format -i ./your-dir-path
```

在新目录中保存格式化后的配置文件：

```bash
./nginx-formatter format -i ./your-dir-path -o ./your-output-dir
```

格式化单个文件：当 `--input` 指向文件时，仅格式化该文件（不再限制 `.conf` 后缀，任意后缀均可）。此时 `--output` 有三种语义：

- 为空：原地覆盖输入文件
- 为已存在的目录：写入 `<输出目录>/<原文件名>`
- 其他情况：视为目标文件路径，必要时自动创建其父目录

```bash
# 原地覆盖
./nginx-formatter format -i ./nginx.conf

# 写入已存在的目录
./nginx-formatter format -i ./nginx.conf -o ./dist

# 写入指定文件路径
./nginx-formatter format -i ./nginx.conf -o ./dist/nginx.formatted.conf
```

### WebUI 用法

启动 WebUI 界面：

```bash
./nginx-formatter serve
```

指定服务端口：

```bash
./nginx-formatter serve -p 8123
```

`--indent` 与 `--char` 用于设置 WebUI 格式化时采用的默认缩进：

```bash
./nginx-formatter serve -p 8123 -n 4 -c space
```

### 查看版本

打印版本号：

```bash
./nginx-formatter version
```

### 向后兼容（旧参数）

旧式的单横线长参数依然可以按原有方式使用：

```bash
# 格式化目录（旧参数）
./nginx-formatter -input=./your-dir-path -output=./your-output-dir -indent=4 -char=" "

# 格式化单个文件（旧参数）
./nginx-formatter -input=./nginx.conf

# 启动 WebUI（旧参数）
./nginx-formatter -web -port=8123
```

### Docker 用法

在 Docker 中使用和上面没有什么区别，比如我们启动一个在 Docker 中的 Web UI 格式化工具服务：

```bash
# 新的子命令写法
docker run --rm -it -p 8080:8080 soulteary/nginx-formatter:latest serve

# 旧参数写法依然可用
docker run --rm -it -p 8080:8080 soulteary/nginx-formatter:latest -web
```


如果你希望格式化当前目录的配置，可以通过类似下面的命令，来使用 Docker 中的程序：

```bash
# 新的子命令写法
docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter:latest format -i /app

# 旧参数写法依然可用
docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter:latest -input=/app
```


## 支持的完整参数列表

运行 `nginx-formatter --help` 查看可用命令，或运行 `nginx-formatter <命令> --help` 查看某个命令的参数与示例：

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

`format` 参数：

```bash
  -c, --char string     Indent char (space/tab/\s/\t) (default " ")
  -n, --indent int      Indent size (default 2)
  -i, --input string    Input directory or file (default: current directory)
  -o, --output string   Output directory or file path
```

`serve` 参数：

```bash
  -c, --char string   Default indent char the WebUI applies (space/tab/\s/\t) (default " ")
  -n, --indent int    Default indent size the WebUI applies (default 2)
  -p, --port int      WebUI port (default 8080)
```

## 参与贡献

欢迎参与贡献！请先阅读[贡献指南](CONTRIBUTING_CN.md)，并遵守[行为准则](CODE_OF_CONDUCT.md)。

## 鸣谢

格式化组件

- 2016/06/24 Slomkowski 使用 Python 创建了一个 nginx 配置文件美化器，在 [Apache-2.0 许可] 下发布。
  - https://github.com/1connect/nginx-config-formatter (https://github.com/slomkowski/nginx-config-formatter)
- 2016/08/24 Yosef 在 [Apache-2.0 许可] 下移植了 JavaScript beautifier。
  - https://github.com/vasilevich/nginxbeautifier
- 2023/04/18，soulteary 根据 [Apache-2.0 许可] 简化程序，修复错误，提高运行速度，并允许在 Golang 中运行。
  - https://github.com/soulteary/nginx-formatter
- v2.0.0 起，soulteary 使用原生 Go AST 的 Nginx 解析器重写了格式化器（移除 JavaScript 运行时），在 [Apache-2.0 许可] 下发布。
  - https://github.com/soulteary/nginx-formatter

网络组件

- Fiber，受 Express 启发的 Web 框架，在 [MIT 许可]下发布。
  - https://github.com/gofiber/fiber
- Code Mirror, 浏览器内的编辑器，在 [MIT 许可]下发布。
  - https://github.com/codemirror/codemirror5
