#!/bin/bash

# 快速部署脚本 - 适用于首次部署
# 使用方法: ./quick-deploy.sh

set -e

echo "🚀 Fluent Life 快速部署脚本"
echo "================================"

# 检查是否在项目根目录
if [ ! -f "docker-compose.yml" ]; then
    echo "❌ 错误: 请在项目根目录执行此脚本"
    exit 1
fi

# 检查环境变量文件
if [ ! -f ".env" ]; then
    echo "📝 创建 .env 文件..."
    if [ -f "env.example" ]; then
        cp env.example .env
        echo "⚠️  请编辑 .env 文件设置正确的配置"
        echo "   特别是 DB_PASSWORD 和 JWT_SECRET"
        read -p "按 Enter 继续（建议先编辑 .env 文件）..."
    else
        echo "❌ 未找到 env.example 文件"
        exit 1
    fi
fi

# 加载环境变量
export $(cat .env | grep -v '^#' | xargs)

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose 未安装"
    exit 1
fi

# 开始部署
echo ""
echo "🔨 开始构建和部署..."
echo ""

# 构建镜像
echo "1️⃣ 构建 Docker 镜像..."
docker-compose build

# 启动服务
echo ""
echo "2️⃣ 启动服务..."
docker-compose up -d

# 等待服务启动
echo ""
echo "3️⃣ 等待服务启动（10秒）..."
sleep 10

# 检查状态
echo ""
echo "4️⃣ 检查服务状态..."
docker-compose ps

echo ""
echo "✅ 部署完成！"
echo ""
echo "📋 访问地址:"
echo "   前端: http://$(hostname -I | awk '{print $1}')"
echo "   后端: http://$(hostname -I | awk '{print $1}'):8081"
echo ""
echo "📊 查看日志: docker-compose logs -f"
echo "🛑 停止服务: docker-compose down"
echo ""


