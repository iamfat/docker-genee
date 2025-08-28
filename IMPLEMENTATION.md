# Docker Genee CLI 插件实现说明

## 概述

这是一个Docker CLI插件，用于操作基理科技的私有镜像仓库 `docker.genee.cn`。插件实现了完整的认证机制，能够从Docker的凭证存储中获取认证信息，并支持两种调用方式。

## 核心功能

### 1. 认证机制

插件实现了多层次的认证信息获取策略：

#### 优先级顺序
1. **Docker凭证助手** - 使用标准的Docker凭证助手
   - `docker-credential-helper`
   - `docker-credential-desktop`
   - `docker-credential-ecr-login`

2. **Docker配置文件** - 从 `~/.docker/config.json` 读取
   - 支持base64编码的认证信息
   - 兼容Docker标准格式

3. **本地凭证文件** - 从 `~/.docker-genee/credentials.json` 读取
   - 向后兼容
   - 作为备用方案

#### 认证流程
```
用户执行 docker genee login
    ↓
验证用户名密码
    ↓
保存到Docker凭证存储
    ↓
同时保存到本地文件（向后兼容）
```

### 2. 命令实现

#### `docker genee login`
- 交互式输入用户名密码
- 验证认证信息
- 保存到Docker凭证存储
- 支持向后兼容

#### `docker genee images`
- 自动获取认证信息
- 显示镜像列表
- 包含仓库、标签、摘要、平台、大小、创建时间
- 支持平台过滤

#### `docker genee search`
- 支持通配符搜索
- 平台限制功能
- 结果数量限制
- 自动认证检查
- 支持标签模式搜索（如：`php:a*`）

### 3. 技术架构

#### 包结构
```
cmd/                    # 命令行实现
├── root.go           # 根命令和全局配置
├── login.go          # 登录命令
├── images.go         # 镜像列表命令
├── search.go         # 搜索命令
└── metadata.go       # Docker CLI插件元数据命令

internal/              # 内部包
└── registry/         # Registry客户端
    ├── client.go     # 核心客户端实现
    └── client_test.go # 测试文件
```

#### 核心组件

**Registry Client**
- 认证信息管理
- HTTP客户端封装
- 多种凭证获取策略
- 错误处理和重试机制

**认证管理器**
- 凭证存储抽象
- 多种存储后端支持
- 向后兼容性保证

**CLI插件支持**
- 支持两种调用方式
- Docker CLI插件元数据
- 命令结构优化

## 实现细节

### 1. 双重调用支持

插件支持两种调用方式：

```go
// 方式1: 直接调用
./docker-genee search node

// 方式2: Docker CLI插件调用
docker genee search node
```

通过将子命令同时添加到 `rootCmd` 和 `geneeCmd` 实现：

```go
func init() {
    rootCmd.AddCommand(searchCmd)      // 支持直接调用
    geneeCmd.AddCommand(searchCmd)     // 支持 docker genee 调用
}
```

### 2. 凭证获取策略

```go
func (c *Client) LoadCredentials() error {
    // 1. 尝试从Docker凭证存储获取
    creds, err := c.getDockerCredentials()
    if err == nil {
        c.credentials = creds
        return nil
    }
    
    // 2. 回退到本地文件
    return c.loadLocalCredentials()
}
```

### 3. 多后端凭证存储

```go
func (c *Client) getDockerCredentials() (*Credentials, error) {
    // 方法1: 凭证助手
    if creds, err := c.getCredentialsFromHelper(); err == nil {
        return creds, nil
    }
    
    // 方法2: Docker配置文件
    if creds, err := c.getCredentialsFromDockerConfig(); err == nil {
        return creds, nil
    }
    
    return nil, fmt.Errorf("无法从Docker获取凭证")
}
```

### 4. 认证状态检查

```go
func (c *Client) HasValidCredentials() bool {
    if err := c.LoadCredentials(); err == nil && c.credentials != nil {
        return true
    }
    return false
}
```

### 5. Docker CLI插件元数据

```go
func runMetadata(cmd *cobra.Command, args []string) error {
    metadata := Metadata{
        SchemaVersion:    "0.1.0",
        Vendor:          "基理科技",
        Version:          Version,
        ShortDescription: "基理科技镜像源操作插件",
        URL:              "https://github.com/iamfat/docker-genee",
    }
    
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", " ")
    return encoder.Encode(metadata)
}
```

## 测试验证

### 单元测试
- 客户端创建和配置
- 凭证数据结构
- 认证状态检查
- 错误处理

### 集成测试
- 真实registry认证
- Docker凭证存储集成
- 命令执行流程
- 双重调用方式验证

### 测试结果
```
=== RUN   TestRealRegistryCredentials
    client_test.go:106: 成功从Docker获取认证信息，用户: iamfat
--- PASS: TestRealRegistryCredentials (0.06s)
```

## 部署和使用

### 构建
```bash
# 开发模式
make dev

# 安装插件
make install
```

### 使用
```bash
# 登录
docker genee login

# 查看镜像
docker genee images
docker genee images --platform arm64

# 搜索镜像
docker genee search php*
docker genee search php:a* --platform arm64
```

### 本地测试
```bash
# 直接调用
./docker-genee search node

# 模拟Docker CLI插件调用
./docker-genee genee search node
```

## 优势特性

1. **无缝集成** - 使用Docker标准凭证存储
2. **向后兼容** - 支持本地凭证文件
3. **多后端支持** - 多种凭证获取方式
4. **双重调用** - 支持直接调用和Docker CLI插件调用
5. **错误处理** - 完善的错误处理和用户提示
6. **测试覆盖** - 完整的测试套件
7. **文档完善** - 详细的使用说明和故障排除

## 未来改进

1. **缓存机制** - 添加认证信息缓存
2. **重试策略** - 网络请求重试机制
3. **监控日志** - 操作日志记录
4. **配置管理** - 更灵活的配置选项
5. **API扩展** - 支持更多registry操作
6. **性能优化** - 优化大量镜像的获取和显示
