#!/bin/bash

# 快速更新前端（仅重新构建前端，不停止其他服务）
# 使用方法: ./quick-update-frontend.sh

set -e

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 快速更新前端...${NC}"

# 检查 docker compose 命令
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
  COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
  COMPOSE_CMD="docker-compose"
else
  echo "❌ Docker Compose 未安装"
  exit 1
fi

echo -e "${YELLOW}🔨 重新构建前端镜像...${NC}"
$COMPOSE_CMD build --no-cache frontend

echo -e "${YELLOW}🔄 重启前端容器...${NC}"
$COMPOSE_CMD up -d --force-recreate frontend

echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
sleep 5

echo -e "${YELLOW}📊 检查状态...${NC}"
$COMPOSE_CMD ps frontend

echo -e "${GREEN}✅ 前端更新完成！${NC}"
echo ""
echo "访问: http://$(hostname -I | awk '{print $1}')"
echo "如果页面没有更新，请清除浏览器缓存"


