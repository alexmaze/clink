# clink 配置分发工具

`clink` 是一个面向 Linux / macOS 的配置文件分发工具。它用单一 `config.yaml` 描述本地软链接、本地复制和远端 SSH 上传，并围绕这套配置提供统一的执行、校验、还原与纳管能力。

本版本做了完整重构，新的产品约束如下：

- 仅支持 Linux / macOS
- 命令体系统一为子命令
- 运行时输出与交互全部英文
- 不兼容历史版本的备份格式与旧用法

## 安装

```sh
go install github.com/alexmaze/clink@latest
```

或使用：

```sh
make build
```

## 命令概览

```sh
clink apply   -c <config.yaml>
clink check   -c <config.yaml>
clink restore -c <config.snapshot.yaml>
clink add     -c <config.yaml> <source>
clink version
```

通用参数：

- `-c, --config`：指定配置文件路径
- `-r, --rule`：按名称或 1-based 序号筛选 rule，可重复传入
- `-d, --dry-run`：只生成计划，不执行
- `-y, --yes`：跳过确认
- `--non-interactive`：禁用交互式输入
- `--output text|json`：选择文本或 JSON 输出

## 配置文件

示例：

```yaml
mode: symlink

hooks:
  pre: echo "start apply"
  post: echo "finish apply"

ssh_servers:
  prod:
    host: 192.168.1.10
    port: 22
    user: root
    key: ~/.ssh/id_rsa

vars:
  APP_HOME: /opt/myapp

rules:
  - name: shell
    items:
      - src: ./dotfiles/.zshrc
        dest: ~/.zshrc

  - name: app-config
    mode: copy
    items:
      - src: ./app/config.yaml
        dest: ${APP_HOME}/config.yaml

  - name: remote-nginx
    mode: ssh
    ssh: prod
    items:
      - src: ./nginx/nginx.conf
        dest: /etc/nginx/nginx.conf
```

说明：

- 顶层 `mode` 是默认分发模式，默认值为 `symlink`
- `rules[].mode` 可覆盖顶层默认值
- 本地 `src` 支持相对路径、绝对路径和 `~/`
- 本地 `dest` 支持相对路径、绝对路径和 `~/`
- SSH 模式下 `dest` 必须是远端绝对路径
- 变量语法为 `${VAR_NAME}`
- `ssh_servers` 中若不提供 `key` 和 `password`，运行时会交互式要求输入密码

## apply

`apply` 根据配置生成执行计划，先备份目标现状，再执行分发。

```sh
clink apply -c ./config.yaml
clink apply -c ./config.yaml -r shell
clink apply -c ./config.yaml -d
clink apply -c ./config.yaml --output json
```

行为说明：

- 本地模式会把原目标备份到 `~/.clink/<timestamp>/payload/...`
- 远端模式会先下载远端原文件到同一备份目录
- 本次执行会写入：
  - `config.snapshot.yaml`
  - `manifest.json`
  - `payload/`

## check

`check` 根据配置校验当前状态。

```sh
clink check -c ./config.yaml
clink check -c ./config.yaml -r 1
```

校验语义：

- `symlink`：检查是否为符号链接，且指向期望源路径
- `copy`：检查目标是否存在，且内容哈希与源一致
- `ssh`：检查远端目标是否存在，且文件类型与本地源匹配

只要存在失败项，命令退出码即非 0。

## restore

`restore` 使用新版本备份目录中的 `manifest.json` 进行还原。

```sh
clink restore
clink restore --backup 20260406_120000
clink restore --backup ~/.clink/20260406_120000 -d
```

行为说明：

- 交互模式下，不指定 `--backup` 会弹出备份选择列表
- 非交互模式下，不指定 `--backup` 默认选择最新备份
- 本地还原使用临时路径后原子替换
- SSH 还原按服务器分组复用连接
- 不兼容历史版本备份目录；缺少 `manifest.json` 的备份无法恢复

## add

`add` 用于把已有文件纳入 `clink` 管理。

```sh
clink add -c ./config.yaml ~/.vimrc
clink add -c ./config.yaml --name shell --mode symlink ~/.zshrc
clink add -c ./config.yaml --rule shell --dest ~/.bashrc ./local/.bashrc
```

行为说明：

- 外部源文件会被复制到 `<config-dir>/.clink/sources/<rule-slug>/`
- 新增 rule 时默认模式为 `symlink`
- 只能追加到本地 rule，不能追加到 SSH rule
- `dry-run` 仅展示计划，不修改文件和配置

## 备份结构

示例：

```text
~/.clink/
  20260406_120000/
    config.snapshot.yaml
    manifest.json
    payload/
      shell/
        home/alex/.zshrc
      remote-nginx/
        etc/nginx/nginx.conf
```

`manifest.json` 会显式记录每个备份条目的：

- rule 名称
- mode
- source
- destination
- 备份文件路径
- SSH server
- path kind
- 备份内容哈希

## 开发

常用命令：

```sh
make build
make test
make build-all
```

说明：

- `make build-all` 仅构建 Linux / macOS 产物
- 当前项目已经不考虑 Windows 支持
- 运行时输出保持英文；新增代码注释和文档请继续遵循仓库规范

## 兼容性说明

本版本不兼容旧版本：

- 不兼容旧命令入口
- 不兼容旧备份目录结构
- 不兼容历史交互习惯

升级时请重新生成备份，并按新命令体系使用。
