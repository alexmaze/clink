# clink 配置管理器


使用 `clink` 可以方便的把你的配置文件集中保存，只需要在 `config.yaml` 文件中定义文件需要分发的目的地，`clink` 就可以帮你将配置文件通过软链、复制或 SSH 上传等方式部署到指定位置，并将原文件备份起来。

集中存放配置文件可以让配置保存、同步更加方便，例如你可以将配置文件目录通过 `dropbox`、 `百度网盘` 等工具在多设备之间进行同步，重装电脑后也只需要下载配置文件目录后通过 `clink` 一键将配置文件应用到新的系统里。

### 使用方式

```sh
go install github.com/alexmaze/clink

# then
clink -c <配置文件目录>/config.yaml
```

### 功能

- [x] 通过 `config.yaml` 配置文件指定配置文件位置
- [x] 自动备份原始文件
- [x] 支持变量，可以在 rules 的路径定义中使用变量
- [x] 规则执行前后增加脚本 Hook 功能（例如安装软件等）
- [x] 增加配置文件分发模式：symlink（软链接）/ copy（本地复制）/ ssh（远程 SFTP 上传）
- [ ] 选择历史备份进行还原

### config.yaml

```yaml
mode: symlink   # 全局默认模式（可选，默认 symlink；可选值：symlink / copy / ssh）

hooks:           # 顶层 hook，在所有规则执行前/后运行
  pre: echo 'start'
  post: echo 'all done'

ssh_servers:    # SSH 服务器定义（ssh 模式使用）
  my-server:
    host: 192.168.1.1
    port: 22          # 默认 22
    user: root
    key: ~/.ssh/id_rsa   # key 与 password 二选一；都不填则运行时 prompt 输入密码
    # password: secret

vars:
  V2RAY: /etc/v2ray

rules:
  - name: vim 配置
    # mode 不写则继承全局（此处为 symlink）
    hooks:       # rule 级别 hook，在该规则执行前/后运行
      pre: brew install vim
      post: echo 'vim ready'
    items:                      # 配置文件（夹）列表
      - src: .src/.vimrc        # 可以使用相对路径，起始路径为 yaml 文件所在目录
        dest: /root/.vimrc      # 可以使用绝对路径
      - src: ./.vim/autoload    # 可以指定文件夹，不存在的文件夹会自动创建
        dest: ~/.vim/autoload   # 可以使用 ~ 代表当前用户的 home 目录

  - name: v2ray 配置 (copy 模式)
    mode: copy
    items:
      - src: ./v2ray/config.json
        dest: ${V2RAY}/config.json

  - name: 远程服务器配置 (ssh 模式)
    mode: ssh
    ssh: my-server          # 引用 ssh_servers 中的 key
    items:
      - src: ./dotfiles/.vimrc
        dest: /root/.vimrc  # 远程路径，不做本地路径处理
```

### 分发模式说明

| 模式 | 说明 |
|------|------|
| `symlink`（默认） | 在目标路径创建软链接，指向源文件 |
| `copy` | 将源文件/目录递归复制到目标路径（真实副本）|
| `ssh` | 通过 SFTP 将源文件上传到远程服务器；旧文件会先下载到本地备份目录 |

- `mode` 可在顶层设置全局默认值，rule 级别可单独覆盖
- SSH 鉴权优先使用密钥文件（`key`），其次密码（`password`），都不填则运行前交互式 prompt

Hook 执行顺序：

```
pre hook (全局)
  ↓
[rule 1] pre hook → backup + deploy items → post hook
[rule 2] pre hook → backup + deploy items → post hook
  ↓
post hook (全局)
```

> Hook 命令通过 `sh -c` 执行，若退出码非 0 则**立即中止**整个流程。
