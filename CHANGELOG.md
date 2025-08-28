# 变更日志

本项目遵循 [语义化版本](https://semver.org/lang/zh-CN/) 规范。

## [1.0.1] - 2025-08-28

### 修复
- **OCI 格式 manifest 支持**：修复了对 OCI 格式镜像的支持
  - 支持 `application/vnd.oci.image.index.v1+json` 多架构 manifest
  - 支持 `application/vnd.oci.image.manifest.v1+json` 单架构 manifest
  - 正确解析 OCI 格式的平台信息
  - 修复了 OCI 格式镜像的 DIGEST、SIZE 和 CREATED 信息显示
  - 过滤掉无效的平台信息（`unknown/unknown`）
  - 正确计算 OCI Image Index 的实际镜像大小

### 变更
- 优化了构建流程
- 改进了文档结构
- 改进了平台检测逻辑，支持多种 manifest 格式

## [1.0.0] - 2025-08-28

### 新增
- Docker CLI 插件支持
- 基理科技私有镜像源操作
- 登录功能 (`docker genee login`)
- 镜像列表查看 (`docker genee images`)
- 镜像搜索功能 (`docker genee search`)
- 支持通配符搜索
- 平台过滤功能
- Docker 凭证存储集成
- 多后端认证支持
- 支持多平台编译 (macOS Intel/Apple Silicon, Linux AMD64/ARM64)
- GitHub Actions 自动化构建和发布
- 发布脚本和工具

### 技术特性
- 支持两种调用方式：
  - 直接调用：`./docker-genee search node`
  - Docker CLI 插件调用：`docker genee search node`
- 实现了 `docker-cli-plugin-metadata` 命令
- 完整的认证机制
- 向后兼容的凭证存储

### 架构
- 基于 Cobra 的命令行框架
- 模块化的代码结构
- 完整的测试覆盖
- 跨平台支持
