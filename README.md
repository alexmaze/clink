# config-manager 配置管理器

> 本工具基于 Deno 开发，使用前需要先安装 Deno

### 使用方式

```sh
make install
cm config.yaml

# or

deno run https://<TODO>/cm -A --unstable
```

### 功能

- [x] 通过 `config.yaml` 配置软链配置文件
- [x] 自动备份原始文件
- [ ] 选择备份进行还原
- [ ] 指定备份路径
- [ ] 每个规则增加运行指定 Before & After 脚本功能

### config.yaml

```yaml
rules:
  - name: vim 配置
    files:                      # 文件列表
      - src: ./.vimrc           # 相对路径，起始路径为 config.yaml 文件所在目录
        dist: /home/alex/.vimrc # 绝对路径
    folders:                    # 文件夹列表，不存在的文件夹会自动创建
      - src: ./.vim/autoload
        dist: ~/.vim/autoload   # 支持使用 ~ 代表当前用户的 home 目录
```

