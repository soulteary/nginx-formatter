# Nginx Formatter / Nginx 格式化工具

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) [![Go Report Card](./.github/goreportcard.svg)](./.github/goreportcard-report.md) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

<p style="text-align: center;">
  <a href="README.md" target="_blank">ENGLISH</a> | <a href="README_CN.md">中文文档</a>
</p>

<img src=".github/logo.png" width="120" >

一款 10MB 左右的，小巧、简洁的 Nginx 格式化工具，支持命令行、WebUI、Docker、x86、ARM、macOS、Linux。

<img src=".github/preview.png">

> **[v2.5.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.5.0) 更新说明**
>
> - 修复多处「格式化会静默破坏被覆盖的配置」的问题。裸词中间的引号（`sub_filter href="/old" href="/new";`、`alias /data/o'brien/;`）此前被当作词边界切开，打印时又用空格拼回去，改变了指令的参数个数，导致 nginx 拒绝加载该配置。裸词的终止规则现已与 nginx 自身的分词器一致。
> - 移除了在解析前对整个文件运行的 `return` 正则归一化。它会给本已单引号包裹的参数再加一层引号 —— `return 200 'ok';` 变成 `return 200 "'ok'";`，改变了 nginx 实际返回的内容 —— 并且会改写 `*_by_lua_block` 内容、注释和引号字符串中的 `return` 字样。自 v2.0 起 AST 解析器已使它变得多余。
> - 空块收尾大括号上的注释（`upstream backend {` / `} # TODO`）不再被删除，且格式化现在是幂等的：连续运行两次得到相同的文件。
> - 含有 NUL 字节或非法 UTF-8 的输入现在会被拒绝，而不是被静默截断或用替换字符改写，且原文件保持不变。未闭合的引号、`${` 或行尾反斜杠会报错，而不是每运行一次就补一个 `;`。
> - **行为变更：** `format -i <目录>` 在未指定 `--output` 时改为就地格式化该目录，与单文件模式一致。此前它会把格式化后的文件树写进当前工作目录，导致目标目录纹丝未动、可能覆盖工作目录中的同名文件，并使文档中的两条 Docker 命令成为空操作。
> - **行为变更：** 格式化输出现在以且仅以一个换行结尾。以指令结尾的配置此前会丢失尾随换行，因此首次运行会产生一次性的 diff。
> - 修复文档中列出的缩进字符：`--char '\s'` 此前会把字面量反斜杠-s 写进每一行却回显 `[SPACE]`，而文档同样列出的 `--char '\t'` 会被直接拒绝。两者现在都会解析为真实的空格和制表符。
> - 符号链接会被跳过并给出提示，而不是跟随，因此常见的 `sites-enabled` → `sites-available` 布局只会通过真实文件格式化一次，而不会让整次运行中止。
> - 文件改为原子写入（临时文件 + rename），并保留原有的权限位与属主，不再就地截断并强制设为 0600。单个无法解析的文件也不再中断整批处理。
> - 新增 `serve --host` 用于限制 WebUI 的监听地址，并加入 1 MiB 请求体上限和 64 层嵌套上限 —— 缩进量随嵌套深度平方增长，此前几 KB 的嵌套块即可耗尽内存。格式化失败现在会显示解析器的错误信息和行号，而不是笼统的 `format error`。
>
> **[v2.4.0](https://github.com/soulteary/nginx-formatter/releases/tag/v2.4.0) 更新说明**
>
> - 修复 WebUI 静默吞掉 Nginx 变量的问题。格式化结果此前通过 `regexp.ReplaceAllString` 拼接进页面，而替换串中的 `$name` 会被当作捕获组引用，导致 `$host`、`$remote_addr`、`$upstream_addr` 被替换为空字符串。
> - WebUI 现在会对格式化结果做 HTML 转义，配置中包含 `</textarea>` 或 `<` 不再会破坏页面或注入标记。
> - 格式化结果改为由 `POST /format` 直接返回，不再存入进程级缓存后经重定向取回，因此不会把某位访问者的配置展示给另一位。`GET /` 始终返回默认页面。
> - 将 `gosec` 安全扫描固定到发布版本的提交 SHA，不再跟踪 `master` 分支。
>
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
docker pull soulteary/nginx-formatter:v2.5.0
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

### GitHub Actions

如需在 Pull Request 中自动检查 Nginx 配置格式，可以使用 [soulteary/nginx-format-action](https://github.com/soulteary/nginx-format-action)。创建 `.github/workflows/nginx-format.yml`：

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

关于写入模式、缩进设置、版本固定及更多示例，请查看 [Nginx Format Action 中文文档](https://github.com/soulteary/nginx-format-action/blob/main/README_CN.md)。

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

格式化指定目录中的配置文件。未指定 `--output` 时会就地格式化，与单文件模式一致：

```bash
./nginx-formatter format -i ./your-dir-path
```

符号链接会被跳过并给出提示，而不是跟随。因此常见的 `sites-enabled` → `sites-available`
布局只会通过真实文件格式化一次，链接结构保持不变。

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

WebUI 默认监听所有网卡，以便下文的 Docker 用法正常工作。可用 `--host` 限制为仅本机访问：

```bash
./nginx-formatter serve --host 127.0.0.1
```

### CI：只检查不写入

`--check` 会列出未格式化的文件，只要有就以 1 退出；`--diff` 则打印将要发生变更的
unified diff。两者都不写任何文件，且 stdout 上只有这部分输出，可以直接管道使用：

```bash
# 只要有文件未格式化就让构建失败
./nginx-formatter format -i ./conf.d --check

# 查看具体会改动什么
./nginx-formatter format -i ./conf.d --diff
```

### 标准输入 / 标准输出

`--input -` 从 stdin 读取配置并把结果写到 stdout，这正是编辑器「保存时格式化」所需的形式：

```bash
cat nginx.conf | ./nginx-formatter format -i -
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
      --check           Do not write; list files that are not formatted and exit 1 if any
      --diff            Do not write; print a unified diff of what would change and exit 1 if any
  -n, --indent int      Indent size (default 2)
  -i, --input string    Input directory or file, or "-" for stdin (default: current directory)
  -o, --output string   Output directory or file path
```

`serve` 参数：

```bash
  -c, --char string   Default indent char the WebUI applies (space/tab/\s/\t) (default " ")
      --host string   Address to bind (default: all interfaces)
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
