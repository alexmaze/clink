# clink 配置管理器


使用 `clink` 可以方便的把你的配置文件集中保存，只需要在 `config.yaml` 文件中定义文件需要软链的目的地，`clink` 就可以帮你将配置文件软链到指定的位置，并将原文件备份起来。

集中存放配置文件可以让配置保存、同步更加方便，例如你可以将配置文件目录通过 `dropbox`、 `百度网盘` 等工具在多设备之间进行同步，重装电脑后也只需要下载配置文件目录后通过 `clink` 一键将配置文件应用到新的系统里。

### 使用方式

```sh
go install github.com/alexmaze/clink

# or 
git clone git@github.com:alexmaze/clink.git
cd clink && make install

# then
clink <配置文件目录>/config.yaml
```

### 功能

- [x] 通过 `config.yaml` 配置文件指定配置文件位置
- [x] 自动备份原始文件
- [x] 支持变量，可以在 rules 的路径定义中使用变量
- [ ] 增加配置文件分发模式：（本地/远程）复制模式（当前是软连接）
- [ ] 选择历史备份进行还原
- [ ] 规则执行前后增加脚本 Hook 功能（例如安装软件等）

### config.yaml

```yaml
vars:
  V2RAY: /etc/v2ray

rules:
  - name: vim 配置
    items:                      # 配置文件（夹）列表
      - src: .src/.vimrc        # 可以使用相对路径，起始路径为 yaml 文件所在目录
        dest: /root/.vimrc      # 可以使用绝对路径
      - src: ./.vim/autoload    # 可以指定文件夹，不存在的文件夹会自动创建
        dest: ~/.vim/autoload   # 可以使用 ~ 代表当前用户的 home 目录
  - name: v2ray 配置
    items:
      - src: ./v2ray/config.json
        dest: ${V2RAY}/config.json
```
