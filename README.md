# config-manager 配置管理器

> 本工具基于 Deno 开发，使用前需要先安装 Deno https://deno.land/#installation

使用 `config-manager` 可以方便的把你的配置文件集中保存，只需要在 `config.yaml` 文件中定义文件需要软链的目的地，`config-manager` 就可以帮你将配置文件软链到指定的位置，并将原文件备份起来。

集中存放配置文件可以让配置保存、同步更加方便，例如你可以将配置文件目录通过 `dropbox`、 `百度网盘` 等工具在多设备之间进行同步，重装电脑后也只需要下载配置文件目录后通过 `config-manager` 一键将配置文件应用到新的系统里。

### 使用方式

```sh
git clone git@github.com:alexmaze/config-manager.git
cd config-manager && make install
cm <配置文件目录>/config.yaml

# or

deno run -A --unstable https://alexyan.cc/deno/cm.js <配置文件目录>/config.yaml
```

### 功能

- [x] 通过 `config.yaml` 配置软链配置文件
- [x] 自动备份原始文件
- [ ] 增加配置文件分发模式：复制模式（当前是软连接）
- [ ] 选择历史备份进行还原
- [ ] 规则执行前后增加脚本 Hook 功能（例如安装软件等）

### config.yaml

```yaml
rules:
  - name: vim 配置
    files:                      # <可选>文件列表
      - src: ./.vimrc           # 可以使用相对路径，起始路径为 config.yaml 文件所在目录
        dist: /home/alex/.vimrc # 可以使用绝对路径
    folders:                    # <可选>文件夹列表，不存在的文件夹会自动创建
      - src: ./.vim/autoload
        dist: ~/.vim/autoload   # 可以使用 ~ 代表当前用户的 home 目录
```
