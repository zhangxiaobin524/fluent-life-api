#!/bin/bash

# 项目上传到服务器的辅助脚本
# 使用方法: ./upload-to-server.sh user@server-ip

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

if [ -z "$1" ]; then
    echo -e "${RED}❌ 请提供服务器地址${NC}"
    echo "使用方法: ./upload-to-server.sh user@server-ip"
    echo "示例: ./upload-to-server.sh root@123.456.789.0"
    exit 1
fi

SERVER=$1
REMOTE_DIR="/opt/fluent-life"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  上传项目到服务器${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}服务器: ${SERVER}${NC}"
echo -e "${YELLOW}目标目录: ${REMOTE_DIR}${NC}"
echo ""

# 检查是否在项目根目录
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}❌ 请在项目根目录执行此脚本${NC}"
    exit 1
fi

# 确认
read -p "确认上传到服务器? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "已取消"
    exit 0
fi

echo ""
echo -e "${YELLOW}[1/3] 创建远程目录...${NC}"
ssh $SERVER "mkdir -p ${REMOTE_DIR}"

echo -e "${YELLOW}[2/3] 上传项目文件...${NC}"
# 使用 rsync 上传（如果可用），否则使用 scp
if command -v rsync &> /dev/null; then
    rsync -avz --progress \
        --exclude 'node_modules' \
        --exclude '.git' \
        --exclude 'dist' \
        --exclude '*.log' \
        --exclude '.env' \
        ./ $SERVER:${REMOTE_DIR}/
else
    echo "使用 scp 上传..."
    scp -r \
        --exclude 'node_modules' \
        --exclude '.git' \
        --exclude 'dist' \
        --exclude '*.log' \
        --exclude '.env' \
        ./* $SERVER:${REMOTE_DIR}/
fi

echo -e "${YELLOW}[3/3] 设置文件权限...${NC}"
ssh $SERVER "chmod +x ${REMOTE_DIR}/*.sh 2>/dev/null || true"

echo ""
echo -e "${GREEN}✅ 上传完成！${NC}"
echo ""
echo -e "${YELLOW}下一步操作：${NC}"
echo "  ssh $SERVER"
echo "  cd ${REMOTE_DIR}"
echo "  cp env.example .env"
echo "  nano .env  # 编辑配置文件"
echo "  ./quick-deploy.sh"
echo ""


