#!/bin/bash

# 快速部署前端代码到服务器
# 使用方法: ./deploy-frontend.sh [服务器IP] [用户名]

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取参数
SERVER_HOST=${1:-"120.55.250.184"}
SERVER_USER=${2:-"root"}

echo -e "${BLUE}🚀 开始部署前端代码到服务器...${NC}"
echo -e "${YELLOW}服务器: ${SERVER_USER}@${SERVER_HOST}${NC}"
echo ""

# 检查是否在正确的目录
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}❌ 请在 fluent-life-api 目录下运行此脚本${NC}"
    exit 1
fi

# 检查 rsync 是否可用
if ! command -v rsync &> /dev/null; then
    echo -e "${YELLOW}⚠️  rsync 未安装，使用 scp 代替${NC}"
    USE_SCP=true
else
    USE_SCP=false
fi

echo -e "${BLUE}📦 步骤 1: 上传前端代码...${NC}"

# 上传前端代码
if [ "$USE_SCP" = true ]; then
    echo "使用 scp 上传..."
    scp -r ../fluent-life-frontend/* ${SERVER_USER}@${SERVER_HOST}:/opt/fluent-life/fluent-life-frontend/
else
    echo "使用 rsync 上传..."
    rsync -avz --exclude 'node_modules' \
        --exclude '.git' \
        --exclude 'dist' \
        --exclude '*.log' \
        --exclude '.env' \
        ../fluent-life-frontend/ ${SERVER_USER}@${SERVER_HOST}:/opt/fluent-life/fluent-life-frontend/
fi

echo -e "${GREEN}✅ 代码上传完成${NC}"
echo ""

echo -e "${BLUE}🔨 步骤 2: 在服务器上重新构建前端...${NC}"

# 在服务器上执行构建和重启
ssh ${SERVER_USER}@${SERVER_HOST} << 'DEPLOY_SCRIPT'
set -e
cd /opt/fluent-life/fluent-life-api

# 检查 docker compose 命令
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
  COMPOSE_CMD="docker-compose"
else
  echo "❌ Docker Compose 未安装"
  exit 1
fi

echo "🛑 停止前端容器..."
$COMPOSE_CMD stop frontend || true

echo "🔨 重新构建前端镜像..."
$COMPOSE_CMD build --no-cache frontend

echo "🚀 启动前端容器..."
$COMPOSE_CMD up -d frontend

echo "⏳ 等待服务启动..."
sleep 5

echo "📊 检查容器状态..."
$COMPOSE_CMD ps frontend

echo "✅ 前端部署完成！"
DEPLOY_SCRIPT

echo ""
echo -e "${GREEN}✅ 部署完成！${NC}"
echo ""
echo "访问地址:"
echo "  前端: http://${SERVER_HOST}"
echo ""
echo "如果页面没有更新，请清除浏览器缓存（Ctrl+Shift+R 或 Cmd+Shift+R）"


