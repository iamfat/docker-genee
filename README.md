# docker-genee

一个Docker CLI插件，实现对基理科技私有镜像源的操作。

## 功能特性

- **登录功能**: 完成私有镜像源的登录
- **查看镜像**: 查看私有镜像列表
- **搜索镜像**: 搜索镜像，支持通配符和平台限制

## 安装方法

### 使用Makefile（推荐）

```bash
# 克隆仓库
git clone https://github.com/iamfat/docker-genee.git
cd docker-genee

# 安装插件
make install
```

### 手动安装

```bash
# 构建插件
go build -o docker-genee .

# 创建插件目录
mkdir -p ~/.docker/cli-plugins

# 复制插件
cp docker-genee ~/.docker/cli-plugins/docker-genee
chmod +x ~/.docker/cli-plugins/docker-genee
```

## 使用方法

### 登录到 docker.genee.cn

```bash
# 登录到私有registry
docker genee login
```

系统会提示输入用户名和密码。

### 查看镜像列表

```bash
# 查看所有镜像
docker genee images

# 限制平台
docker genee images --platform arm64
```

### 搜索镜像

```bash
# 搜索以ph开头的镜像
docker genee search ph*

# 搜索php镜像中以a开头的标签
docker genee search php:a*

# 限制平台为arm64
docker genee search ph* --platform arm64

# 限制搜索结果数量
docker genee search ph* --limit 50
```

## 开发

### 本地开发

```bash
# 下载依赖
make deps

# 开发模式构建
make dev

# 运行本地测试
./docker-genee --help
./docker-genee search node
```

### 代码质量

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test
```

### 清理

```bash
# 清理构建文件
make clean

# 卸载插件
make uninstall
```

## 项目结构

```
docker-genee/
├── cmd/                    # 命令行实现
│   ├── root.go           # 根命令
│   ├── login.go          # 登录命令
│   ├── images.go         # 镜像列表命令
│   ├── search.go         # 搜索命令
│   └── metadata.go       # 插件元数据命令
├── internal/              # 内部包
│   └── registry/         # Registry客户端
│       ├── client.go     # 客户端实现
│       └── client_test.go # 测试文件
├── main.go               # 主程序入口
├── Makefile              # 构建工具
├── go.mod                # Go模块文件
└── README.md             # 项目说明
```

## 认证机制

插件使用Docker的凭证存储系统来管理认证信息：

### 登录流程
1. 使用 `docker genee login` 命令登录
2. 认证信息会保存到Docker的凭证存储中
3. 同时也会保存到本地 `~/.docker-genee/` 目录（向后兼容）

### 认证信息获取
插件在执行非登录操作前，会按以下优先级获取认证信息：
1. **Docker凭证助手** (`docker-credential-helper`, `docker-credential-desktop`, `docker-credential-ecr-login`)
2. **Docker配置文件** (`~/.docker/config.json`)
3. **本地凭证文件** (`~/.docker-genee/credentials.json`) - 向后兼容

### 配置目录
- `~/.docker/cli-plugins/`: Docker CLI插件目录
- `~/.docker-genee/`: 本地凭证存储（向后兼容）
- `~/.docker/config.json`: Docker标准凭证配置

## 故障排除

### 插件无法识别

```bash
# 检查插件是否正确安装
ls -la ~/.docker/cli-plugins/

# 检查插件权限
chmod +x ~/.docker/cli-plugins/docker-genee

# 重新安装
make uninstall
make install
```

### 认证失败

```bash
# 方法1: 重新登录（推荐）
docker genee login

# 方法2: 删除旧的认证信息
rm -rf ~/.docker-genee/
docker genee login

# 方法3: 检查Docker凭证存储
docker-credential-helper list

# 方法4: 检查Docker配置文件
cat ~/.docker/config.json
```

### 无法获取认证信息

如果插件无法从Docker获取认证信息，可能的原因：

1. **Docker未运行**: 确保Docker Desktop正在运行
2. **凭证助手未安装**: 安装相应的凭证助手
3. **权限问题**: 检查Docker配置文件的权限
4. **网络问题**: 确保能够访问私有镜像源

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

MIT License