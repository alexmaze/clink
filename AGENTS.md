# Repository Guidelines

## Project Structure & Module Organization
这是一个 Go CLI 项目，入口在 `main.go`、`check.go`、`restore.go`。配置解析位于 `config/`，可复用能力放在 `lib/`，其中包括 `color`、`fileutil`、`spinner`、`sshutil`、`tui` 等子模块。测试文件与源码同目录放置，命名为 `*_test.go`，如 `config/config_test.go`、`lib/color/color_test.go`。补充说明见 `README.md` 与 `README.zh.md`；容器化测试使用 `Dockerfile.test` 和 `run_tests.sh`。

## Build, Test, and Development Commands
- `make run`：直接运行当前源码，适合本地调试 CLI。
- `make build`：生成本地二进制 `./clink`，并注入版本信息。
- `make test`：执行 `go test github.com/alexmaze/clink/...`。
- `./run_tests.sh`：使用 `podman` 或 `docker` 在容器中跑完整测试并输出覆盖率摘要。
- `make build-all`：构建多平台发布产物到 `dist/`。

## Coding Style & Naming Conventions
提交前保持 `gofmt` 输出，不要手动对齐格式。Go 代码使用 tab 缩进，导出标识符采用 `PascalCase`，未导出标识符采用 `camelCase`。文件名与包名保持小写；测试函数使用 `TestXxx` 命名。新增逻辑优先放入现有包中，避免在根目录继续堆积无关入口文件。新增注释应简短且解释“为什么”，项目当前允许中英文注释并存，但请优先保持与所在文件一致。

## Testing Guidelines
测试基于 Go `testing`，断言库使用 `testify/assert`。变更配置解析、路径处理、恢复流程或终端交互相关代码时，优先补充相邻模块测试。本地至少运行 `make test`；涉及运行环境差异、容器行为或覆盖率回归时，再运行 `./run_tests.sh`。新增测试文件请贴近被测代码放置，覆盖正常路径和失败路径。

## Commit & Pull Request Guidelines
现有历史以 Conventional Commits 为主，例如 `feat: add --check feature`、`fix: fix chinese charactor makes table chaos`。继续使用 `feat:`、`fix:`、`docs:`、`test:` 等前缀，主题句用祈使式并聚焦单一改动。PR 应包含：变更目的、主要实现点、测试结果；若修改 CLI 输出、交互流程或文档示例，请附终端截图或示例命令。

## Security & Configuration Tips
不要在仓库中提交真实 SSH 密钥、密码或生产 `config.yaml`。涉及远端部署时，优先使用示例配置与脱敏数据；文档中的路径示例请使用占位路径，如 `~/.clink/`、`<config-dir>/config.yaml`。
