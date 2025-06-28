#!/bin/bash

# FireMail Docker构建脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置
IMAGE_NAME="luofengyuan/firemailplus"
VERSION="latest"

echo -e "${GREEN}开始构建FireMail Docker镜像...${NC}"

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}错误: Docker未运行或无法访问${NC}"
    exit 1
fi

# 检查必要文件
if [ ! -f "Dockerfile" ]; then
    echo -e "${RED}错误: 未找到Dockerfile${NC}"
    exit 1
fi

if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}错误: 未找到docker-compose.yml${NC}"
    exit 1
fi

# 清理旧的构建缓存（可选）
echo -e "${YELLOW}清理Docker构建缓存...${NC}"
docker builder prune -f > /dev/null 2>&1 || true

# 构建镜像
echo -e "${GREEN}构建Docker镜像: ${IMAGE_NAME}:${VERSION}${NC}"
docker build -t ${IMAGE_NAME}:${VERSION} .

# 检查构建结果
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Docker镜像构建成功!${NC}"
    
    # 显示镜像信息
    echo -e "${YELLOW}镜像信息:${NC}"
    docker images ${IMAGE_NAME}:${VERSION}
    
    # 显示镜像大小
    IMAGE_SIZE=$(docker images ${IMAGE_NAME}:${VERSION} --format "{{.Size}}")
    echo -e "${GREEN}镜像大小: ${IMAGE_SIZE}${NC}"
    
else
    echo -e "${RED}❌ Docker镜像构建失败!${NC}"
    exit 1
fi

echo -e "${GREEN}构建完成! 可以使用以下命令运行:${NC}"
echo -e "${YELLOW}docker-compose up -d${NC}"
