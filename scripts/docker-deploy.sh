#!/bin/bash

# FireMail Docker部署脚本
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
CONTAINER_NAME="firemail-app"
COMPOSE_FILE="docker-compose.yml"

echo -e "${GREEN}开始部署FireMail应用...${NC}"

# 检查Docker和docker-compose
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker未安装${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}错误: docker-compose未安装${NC}"
    exit 1
fi

# 检查配置文件
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}错误: 未找到$COMPOSE_FILE${NC}"
    exit 1
fi

# 停止现有容器
echo -e "${YELLOW}停止现有容器...${NC}"
docker-compose down > /dev/null 2>&1 || true

# 清理旧容器和镜像（可选）
echo -e "${YELLOW}清理旧容器...${NC}"
docker container prune -f > /dev/null 2>&1 || true

# 启动服务
echo -e "${GREEN}启动FireMail服务...${NC}"
docker-compose up -d

# 等待服务启动
echo -e "${YELLOW}等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${BLUE}检查服务状态...${NC}"
docker-compose ps

# 检查健康状态
echo -e "${YELLOW}等待健康检查...${NC}"
for i in {1..30}; do
    if docker-compose ps | grep -q "healthy"; then
        echo -e "${GREEN}✅ 服务健康检查通过!${NC}"
        break
    elif [ $i -eq 30 ]; then
        echo -e "${RED}⚠️  健康检查超时，请检查日志${NC}"
        break
    else
        echo -n "."
        sleep 2
    fi
done

# 显示访问信息
echo -e "${GREEN}🎉 FireMail部署完成!${NC}"
echo -e "${BLUE}访问地址: http://localhost:3000${NC}"
echo -e "${BLUE}默认账户: admin / admin123${NC}"
echo -e "${YELLOW}⚠️  请及时修改默认密码和JWT密钥!${NC}"

# 显示有用的命令
echo -e "${BLUE}常用命令:${NC}"
echo -e "  查看日志: ${YELLOW}docker-compose logs -f${NC}"
echo -e "  停止服务: ${YELLOW}docker-compose down${NC}"
echo -e "  重启服务: ${YELLOW}docker-compose restart${NC}"
echo -e "  查看状态: ${YELLOW}docker-compose ps${NC}"
