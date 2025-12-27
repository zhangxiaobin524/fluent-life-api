#!/bin/bash

# CentOS ECS 服务器初始化脚本
# 用于全新服务器环境准备
# 使用方法: bash init-centos-server.sh

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  CentOS ECS 服务器初始化脚本${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ 请使用 root 用户运行此脚本${NC}"
    echo "使用方法: sudo bash init-centos-server.sh"
    exit 1
fi

# 1. 更新系统
echo -e "${YELLOW}[1/8] 更新系统包...${NC}"
yum update -y
yum install -y epel-release
echo -e "${GREEN}✅ 系统更新完成${NC}"
echo ""

# 2. 安装基础工具
echo -e "${YELLOW}[2/8] 安装基础工具...${NC}"
yum install -y \
    curl \
    wget \
    git \
    vim \
    net-tools \
    unzip \
    zip \
    htop \
    tree \
    telnet \
    nc
echo -e "${GREEN}✅ 基础工具安装完成${NC}"
echo ""

# 3. 安装 Docker
echo -e "${YELLOW}[3/8] 安装 Docker...${NC}"

# 检查 Docker 是否已安装
if command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker 已安装，跳过...${NC}"
else
    # 卸载旧版本
    yum remove -y docker docker-client docker-client-latest docker-common \
        docker-latest docker-latest-logrotate docker-logrotate docker-engine 2>/dev/null || true
    
    # 安装 Docker 仓库
    yum install -y yum-utils
    yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    
    # 安装 Docker Engine
    yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    
    # 启动 Docker
    systemctl start docker
    systemctl enable docker
    
    # 验证安装
    docker --version
    echo -e "${GREEN}✅ Docker 安装完成${NC}"
fi
echo ""

# 4. 安装 Docker Compose (独立版本)
echo -e "${YELLOW}[4/8] 安装 Docker Compose...${NC}"

if command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}⚠️  Docker Compose 已安装，跳过...${NC}"
else
    # 下载最新版本的 Docker Compose
    COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep 'tag_name' | cut -d\" -f4)
    echo "下载 Docker Compose ${COMPOSE_VERSION}..."
    
    curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" \
        -o /usr/local/bin/docker-compose
    
    chmod +x /usr/local/bin/docker-compose
    
    # 创建软链接（兼容性）
    ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    
    # 验证安装
    docker-compose --version
    echo -e "${GREEN}✅ Docker Compose 安装完成${NC}"
fi
echo ""

# 5. 配置防火墙
echo -e "${YELLOW}[5/8] 配置防火墙...${NC}"

# 检查防火墙服务
if systemctl is-active --quiet firewalld; then
    echo "配置 firewalld..."
    firewall-cmd --permanent --add-port=80/tcp
    firewall-cmd --permanent --add-port=443/tcp
    firewall-cmd --permanent --add-port=8081/tcp
    firewall-cmd --permanent --add-port=22/tcp
    firewall-cmd --reload
    echo -e "${GREEN}✅ 防火墙配置完成${NC}"
elif systemctl is-active --quiet iptables; then
    echo "配置 iptables..."
    iptables -I INPUT -p tcp --dport 80 -j ACCEPT
    iptables -I INPUT -p tcp --dport 443 -j ACCEPT
    iptables -I INPUT -p tcp --dport 8081 -j ACCEPT
    iptables -I INPUT -p tcp --dport 22 -j ACCEPT
    service iptables save 2>/dev/null || true
    echo -e "${GREEN}✅ 防火墙配置完成${NC}"
else
    echo -e "${YELLOW}⚠️  未检测到防火墙服务，跳过配置${NC}"
    echo -e "${YELLOW}   请确保在 ECS 安全组中开放以下端口：${NC}"
    echo -e "${YELLOW}   - 80 (HTTP)${NC}"
    echo -e "${YELLOW}   - 443 (HTTPS)${NC}"
    echo -e "${YELLOW}   - 8081 (后端 API)${NC}"
    echo -e "${YELLOW}   - 22 (SSH)${NC}"
fi
echo ""

# 6. 优化系统配置
echo -e "${YELLOW}[6/8] 优化系统配置...${NC}"

# 增加文件描述符限制
cat >> /etc/security/limits.conf << EOF
* soft nofile 65535
* hard nofile 65535
EOF

# 优化内核参数
cat >> /etc/sysctl.conf << EOF
# Docker 优化
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1

# 网络优化
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_keepalive_time = 300
EOF

sysctl -p > /dev/null 2>&1 || true

echo -e "${GREEN}✅ 系统配置优化完成${NC}"
echo ""

# 7. 创建项目目录
echo -e "${YELLOW}[7/8] 创建项目目录...${NC}"
PROJECT_DIR="/opt/fluent-life"
mkdir -p $PROJECT_DIR
echo -e "${GREEN}✅ 项目目录已创建: ${PROJECT_DIR}${NC}"
echo ""

# 8. 安装完成总结
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}✅ 服务器初始化完成！${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}已安装的软件：${NC}"
echo "  - Docker: $(docker --version 2>/dev/null | cut -d' ' -f3 || echo '未安装')"
echo "  - Docker Compose: $(docker-compose --version 2>/dev/null | cut -d' ' -f4 || echo '未安装')"
echo "  - Git: $(git --version 2>/dev/null | cut -d' ' -f3 || echo '未安装')"
echo ""
echo -e "${YELLOW}已开放的端口：${NC}"
echo "  - 80 (HTTP)"
echo "  - 443 (HTTPS)"
echo "  - 8081 (后端 API)"
echo "  - 22 (SSH)"
echo ""
echo -e "${YELLOW}项目目录：${NC}"
echo "  ${PROJECT_DIR}"
echo ""
echo -e "${YELLOW}下一步操作：${NC}"
echo "  1. 将项目文件上传到 ${PROJECT_DIR}"
echo "  2. 进入项目目录: cd ${PROJECT_DIR}"
echo "  3. 配置环境变量: cp env.example .env && nano .env"
echo "  4. 执行部署: chmod +x quick-deploy.sh && ./quick-deploy.sh"
echo ""
echo -e "${BLUE}========================================${NC}"


