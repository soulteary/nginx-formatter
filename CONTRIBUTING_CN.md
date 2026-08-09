# 贡献指南 / Contributing

感谢你抽出时间参与贡献！本项目是一款用 Go 编写的、小巧且快速的 Nginx 配置格式化工具。我们欢迎各种形式的贡献：问题反馈、功能建议、文档完善以及代码提交。

<p style="text-align: center;">
  <a href="CONTRIBUTING.md" target="_blank">ENGLISH</a> | <a href="CONTRIBUTING_CN.md">中文</a>
</p>

## 行为准则

参与本项目的所有人都需要遵守我们的[行为准则](CODE_OF_CONDUCT.md)。参与即代表你愿意遵守它。如遇到不当行为，请通过提交 issue 反馈。

## 贡献方式

- **反馈 Bug**：请提交一个 [issue](https://github.com/soulteary/nginx-formatter/issues)，清晰地描述问题；最好附上一份能够复现问题的最小化 Nginx 配置（输入内容 + 期望输出与实际输出的对比）。
- **功能建议**：提交 issue，说明你的使用场景以及该功能带来的价值。
- **完善文档**：修正错别字、优化措辞或补充翻译。请保持 `README.md` 与 `README_CN.md` 内容同步。
- **提交代码**：通过 Pull Request 修复 Bug 或实现新功能（详见下文）。

## 开发环境准备

### 依赖要求

- [Go 1.26+](https://go.dev/dl/)（所需版本已在 [`go.mod`](go.mod) 中固定）
- Git
- 可选：Docker，如果你需要测试容器镜像

### 快速上手

```bash
# 克隆你 fork 的仓库
git clone https://github.com/<你的用户名>/nginx-formatter.git
cd nginx-formatter

# 下载依赖
go mod download

# 编译二进制文件
go build -o nginx-formatter .

# 运行
./nginx-formatter -input=./your-dir-path
```

### 本地启动 WebUI

```bash
go run . -web -port=8080
```

然后在浏览器中打开 http://localhost:8080 。

## 项目结构

```
main.go               程序入口（参数解析 + 分发）
internal/
  checker/            运行环境 / 输入校验
  cmd/                命令行参数解析
  define/             共享常量与默认值
  formatter/          格式化的高层入口
  nginx/              原生 Go AST 的 Nginx 解析器
    lexer.go          词法分析器
    parser.go         语法分析器（token -> AST）
    ast.go            AST 节点定义
    printer.go        AST -> 格式化输出
    token.go          Token 定义
  server/             基于 Fiber 的 WebUI（含内嵌静态资源）
  updater/            读取/写入目录中的配置文件
  version/            版本号
docker/               Dockerfile
.github/workflows/    CI：覆盖率、安全扫描、发布
```

大部分格式化逻辑都在 `internal/nginx/` 中。如果你要修复格式化相关的 Bug，通常应从这里入手。

## 测试

提交 Pull Request 前，请先运行完整的测试套件：

```bash
go test ./...
```

带覆盖率运行测试（与 CI 保持一致）：

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -html=coverage.out
```

如果你修复了 Bug 或新增了功能，请补充相应的测试用例。`internal/nginx/nginx_test.go` 与 `internal/formatter/formatter_test.go` 中的解析器与格式化测试是很好的参考，它们采用了本项目常用的表驱动（table-driven）风格。

## 代码风格

- 提交前请使用 `gofmt` 格式化所有代码：

```bash
gofmt -w .
```

- 运行 `go vet ./...` 以发现常见错误。
- 遵循标准的 Go 编码规范，改动尽量聚焦、精简。
- 仅在解释非显而易见的意图时添加注释，避免写只是复述代码的注释。

## Pull Request 流程

1. Fork 仓库，并基于 `main` 创建主题分支：

```bash
git checkout -b fix/some-bug
```

2. 完成修改、补充测试，并确保全部通过：

```bash
gofmt -w .
go vet ./...
go test ./...
```

3. 使用清晰、有描述性的提交信息，并保持提交在逻辑上聚焦。
4. 推送分支并向 `main` 发起 Pull Request。
5. 在 PR 描述中说明**改了什么**以及**为什么这么改**，并关联相关 issue（例如 `Closes #123`）。
6. 确保 CI 通过。维护者会评审你的 PR，并可能提出修改建议。

## 报告安全问题

安全相关的问题，请按照 [`SECURITY.md`](SECURITY.md) 中描述的流程处理。

## 许可协议

提交贡献即表示你同意你的贡献将采用与本项目相同的许可协议（[Apache-2.0](LICENSE)）授权。
