#!/bin/bash

# 部署脚本 - 用于 ECS 服务器部署
# 使用方法: ./deploy.sh

set -e

echo "🚀 开始部署 Fluent Life 应用..."

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker 未安装，请先安装 Docker${NC}"
    exit 1
fi

# 检查 Docker Compose 是否安装（优先使用 docker compose v2）
if command -v docker &> /dev/null && docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    echo -e "${RED}❌ Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

# 检查环境变量文件
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  未找到 .env 文件，正在创建示例文件...${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env
        # 自动检测服务器IP并设置 VITE_API_BASE_URL
        SERVER_IP=$(hostname -I | awk '{print $1}' || curl -s ifconfig.me)
        if [ -n "$SERVER_IP" ]; then
            sed -i "s|VITE_API_BASE_URL=.*|VITE_API_BASE_URL=http://${SERVER_IP}:8081/api/v1|g" .env
            echo -e "${GREEN}✅ 已自动设置 VITE_API_BASE_URL=http://${SERVER_IP}:8081/api/v1${NC}"
        fi
    else
        echo -e "${RED}❌ .env.example 文件不存在${NC}"
        exit 1
    fi
    echo -e "${YELLOW}⚠️  请检查 .env 文件中的配置是否正确${NC}"
fi

# 检查 VITE_API_BASE_URL 是否包含占位符
if grep -q "your-domain.com" .env 2>/dev/null; then
    echo -e "${YELLOW}⚠️  检测到 .env 文件中包含占位符 'your-domain.com'${NC}"
    SERVER_IP=$(hostname -I | awk '{print $1}' || curl -s ifconfig.me)
    if [ -n "$SERVER_IP" ]; then
        read -p "是否自动替换为服务器IP (${SERVER_IP})? (Y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            sed -i "s|VITE_API_BASE_URL=.*your-domain.com.*|VITE_API_BASE_URL=http://${SERVER_IP}:8081/api/v1|g" .env
            echo -e "${GREEN}✅ 已自动更新 VITE_API_BASE_URL=http://${SERVER_IP}:8081/api/v1${NC}"
        else
            echo -e "${YELLOW}⚠️  请手动编辑 .env 文件，修改 VITE_API_BASE_URL${NC}"
            exit 1
        fi
    fi
fi

# 加载环境变量（Docker Compose 会自动读取 .env 文件，但为了确保，我们也导出）
set -a
source .env
set +a

# 停止旧容器
echo -e "${YELLOW}🛑 停止旧容器...${NC}"
$COMPOSE_CMD down || true

# 清理旧镜像（可选）
read -p "是否清理旧镜像？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}🧹 清理旧镜像...${NC}"
    $COMPOSE_CMD down --rmi all || true
fi

# 构建镜像
echo -e "${YELLOW}🔨 构建 Docker 镜像...${NC}"
$COMPOSE_CMD build --no-cache

# 启动服务
echo -e "${YELLOW}🚀 启动服务...${NC}"
$COMPOSE_CMD up -d

# 等待服务启动
echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${YELLOW}📊 检查服务状态...${NC}"
$COMPOSE_CMD ps

# 检查健康状态
echo -e "${YELLOW}🏥 检查健康状态...${NC}"
sleep 5

# 检查后端健康
if curl -f http://localhost:8081/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 后端服务健康${NC}"
else
    echo -e "${RED}❌ 后端服务不健康${NC}"
fi

# 检查前端
if curl -f http://localhost > /dev/null 2>&1; then
    echo -e "${GREEN}✅ 前端服务正常${NC}"
else
    echo -e "${RED}❌ 前端服务异常${NC}"
fi

# 显示日志
echo -e "${GREEN}📋 查看日志: $COMPOSE_CMD logs -f${NC}"
echo -e "${GREEN}✅ 部署完成！${NC}"
echo ""
echo "访问地址:"
echo "  前端: http://$(hostname -I | awk '{print $1}')"
echo "  后端 API: http://$(hostname -I | awk '{print $1}'):8081"
echo "  健康检查: http://$(hostname -I | awk '{print $1}'):8081/health"


