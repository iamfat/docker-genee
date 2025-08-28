#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取版本号
VERSION=${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")}
VERSION=${VERSION#v} # 移除 v 前缀

echo -e "${BLUE}🚀 开始构建 docker-genee v${VERSION} 发布版本${NC}"

# 检查是否在 git 仓库中
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}❌ 错误: 当前目录不是 git 仓库${NC}"
    exit 1
fi

# 检查是否有未提交的更改
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}⚠️  警告: 有未提交的更改${NC}"
    read -p "是否继续? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}❌ 构建已取消${NC}"
        exit 1
    fi
fi

# 创建构建目录
BUILD_DIR="build/v${VERSION}"
mkdir -p "$BUILD_DIR"

echo -e "${BLUE}📁 创建构建目录: $BUILD_DIR${NC}"

# 构建所有平台版本
echo -e "${BLUE}🔨 开始构建多平台版本...${NC}"

# macOS Intel
echo -e "${YELLOW}构建 macOS Intel 版本...${NC}"
GOOS=darwin GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w" -o "$BUILD_DIR/docker-genee-darwin-amd64" .

# macOS Apple Silicon
echo -e "${YELLOW}构建 macOS Apple Silicon 版本...${NC}"
GOOS=darwin GOARCH=arm64 go build -a -installsuffix cgo -ldflags="-s -w" -o "$BUILD_DIR/docker-genee-darwin-arm64" .

# Linux AMD64
echo -e "${YELLOW}构建 Linux AMD64 版本...${NC}"
GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-s -w" -o "$BUILD_DIR/docker-genee-linux-amd64" .

# Linux ARM64
echo -e "${YELLOW}构建 Linux ARM64 版本...${NC}"
GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -ldflags="-s -w" -o "$BUILD_DIR/docker-genee-linux-arm64" .

echo -e "${GREEN}✅ 所有平台版本构建完成！${NC}"

# 显示构建结果
echo -e "${BLUE}📋 构建结果:${NC}"
ls -la "$BUILD_DIR/"

# 计算文件大小
echo -e "${BLUE}📊 文件大小:${NC}"
for file in "$BUILD_DIR"/*; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo "  $(basename "$file"): $size"
    fi
done

# 创建 SHA256 校验和
echo -e "${BLUE}🔐 生成 SHA256 校验和...${NC}"
cd "$BUILD_DIR"
for file in *; do
    if [ -f "$file" ]; then
        shasum -a 256 "$file" > "$file.sha256"
        echo "  生成: $file.sha256"
    fi
done
cd - > /dev/null

# 创建发布说明模板
RELEASE_NOTES="$BUILD_DIR/RELEASE_NOTES.md"
cat > "$RELEASE_NOTES" << EOF
# docker-genee v${VERSION} 发布说明

## 下载

### macOS
- **Intel (x86_64)**: [docker-genee-darwin-amd64](https://github.com/iamfat/docker-genee/releases/download/v${VERSION}/docker-genee-darwin-amd64)
- **Apple Silicon (ARM64)**: [docker-genee-darwin-arm64](https://github.com/iamfat/docker-genee/releases/download/v${VERSION}/docker-genee-darwin-arm64)

### Linux
- **AMD64 (x86_64)**: [docker-genee-linux-amd64](https://github.com/iamfat/docker-genee/releases/download/v${VERSION}/docker-genee-linux-amd64)
- **ARM64**: [docker-genee-linux-arm64](https://github.com/iamfat/docker-genee/releases/download/v${VERSION}/docker-genee-linux-arm64)

## 安装说明

1. 下载对应平台的二进制文件
2. 重命名为 \`docker-genee\`
3. 移动到 \`~/.docker/cli-plugins/\` 目录
4. 设置执行权限: \`chmod +x ~/.docker/cli-plugins/docker-genee\`

## 使用方法

\`\`\`bash
# 查看帮助
docker genee --help

# 登录到私有镜像源
docker genee login

# 查看镜像列表
docker genee images

# 搜索镜像
docker genee search node
\`\`\`

## 变更日志

请查看 [CHANGELOG.md](../../CHANGELOG.md) 了解详细变更。

## 校验和

\`\`\`
$(cd "$BUILD_DIR" && for file in *.sha256; do echo "$(cat "$file")"; done)
\`\`\`
EOF

echo -e "${GREEN}✅ 发布说明已生成: $RELEASE_NOTES${NC}"

# 显示下一步操作
echo -e "${BLUE}🎯 下一步操作:${NC}"
echo -e "1. 检查构建结果: ${GREEN}ls -la $BUILD_DIR/${NC}"
echo -e "2. 测试二进制文件: ${GREEN}cd $BUILD_DIR && ./docker-genee-darwin-amd64 --version${NC}"
echo -e "3. 提交并推送标签: ${GREEN}git tag v${VERSION} && git push origin v${VERSION}${NC}"
echo -e "4. 或者手动上传到 GitHub Releases: ${GREEN}$BUILD_DIR${NC}"

echo -e "${GREEN}🎉 发布版本构建完成！${NC}"
