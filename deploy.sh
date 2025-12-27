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

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

# 检查环境变量文件
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  未找到 .env 文件，正在创建示例文件...${NC}"
    if [ -f .env.example ]; then
        cp .env.example .env
    else
        echo -e "${RED}❌ .env.example 文件不存在${NC}"
        exit 1
    fi
    echo -e "${YELLOW}⚠️  请编辑 .env 文件设置正确的环境变量${NC}"
    exit 1
fi

# 加载环境变量（Docker Compose 会自动读取 .env 文件，但为了确保，我们也导出）
set -a
source .env
set +a

# 停止旧容器
echo -e "${YELLOW}🛑 停止旧容器...${NC}"
docker-compose down || true

# 清理旧镜像（可选）
read -p "是否清理旧镜像？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}🧹 清理旧镜像...${NC}"
    docker-compose down --rmi all || true
fi

# 构建镜像
echo -e "${YELLOW}🔨 构建 Docker 镜像...${NC}"
docker-compose build --no-cache

# 启动服务
echo -e "${YELLOW}🚀 启动服务...${NC}"
docker-compose up -d

# 等待服务启动
echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${YELLOW}📊 检查服务状态...${NC}"
docker-compose ps

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
echo -e "${GREEN}📋 查看日志: docker-compose logs -f${NC}"
echo -e "${GREEN}✅ 部署完成！${NC}"
echo ""
echo "访问地址:"
echo "  前端: http://$(hostname -I | awk '{print $1}')"
echo "  后端 API: http://$(hostname -I | awk '{print $1}'):8081"
echo "  健康检查: http://$(hostname -I | awk '{print $1}'):8081/health"


