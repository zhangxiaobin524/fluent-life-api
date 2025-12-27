# Fluent Life 部署指南

本文档介绍如何将 Fluent Life 应用部署到 ECS（云服务器）。

## 📋 前置要求

1. **ECS 服务器**（推荐配置）：
   - CPU: 2核+
   - 内存: 4GB+
   - 系统: Ubuntu 20.04+ / CentOS 7+ / Alibaba Cloud Linux
   - 磁盘: 20GB+

2. **已安装的软件**：
   - Docker 20.10+
   - Docker Compose 2.0+

3. **网络配置**：
   - 开放端口：80（前端）、8081（后端）、5432（数据库，可选，建议仅内网访问）

## 🚀 快速部署

### 1. 连接服务器

```bash
ssh root@your-ecs-ip
```

### 2. 安装 Docker 和 Docker Compose

#### Ubuntu/Debian:

```bash
# 更新包列表
apt-get update

# 安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# 安装 Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose

# 启动 Docker
systemctl start docker
systemctl enable docker
```

#### CentOS/Alibaba Cloud Linux:

```bash
# 安装 Docker
yum install -y docker
systemctl start docker
systemctl enable docker

# 安装 Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

### 3. 克隆项目

```bash
# 创建项目目录
mkdir -p /opt/fluent-life
cd /opt/fluent-life

# 克隆项目（或上传项目文件）
git clone https://your-repo-url.git .
# 或者使用 scp 上传项目文件
```

### 4. 配置环境变量

```bash
# 复制环境变量示例文件
cp .env.example .env

# 编辑环境变量
nano .env
```

**重要配置项**：

```env
# 数据库密码（必须修改）
DB_PASSWORD=your_secure_password_here

# JWT 密钥（必须修改，至少32个字符）
JWT_SECRET=your-secret-key-change-in-production-min-32-chars

# 前端 API 地址（使用服务器 IP 或域名）
VITE_API_BASE_URL=http://your-domain.com/api/v1
# 或者使用 IP
# VITE_API_BASE_URL=http://your-server-ip:8081/api/v1
```

### 5. 构建和启动

```bash
# 赋予部署脚本执行权限
chmod +x deploy.sh

# 执行部署
./deploy.sh
```

或者手动执行：

```bash
# 构建镜像
docker-compose build

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 6. 验证部署

```bash
# 检查服务状态
docker-compose ps

# 检查后端健康
curl http://localhost:8081/health

# 检查前端
curl http://localhost
```

## 🔧 配置说明

### 端口映射

- **80**: 前端 Nginx 服务
- **8081**: 后端 API 服务
- **5432**: PostgreSQL 数据库（建议仅内网访问）

### 数据持久化

数据库数据存储在 Docker volume `postgres_data` 中，即使容器删除数据也不会丢失。

### 环境变量

所有环境变量在 `.env` 文件中配置，包括：
- 数据库连接信息
- JWT 密钥
- API 地址

## 📝 常用命令

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres
```

### 重启服务

```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart backend
docker-compose restart frontend
```

### 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用）
docker-compose down -v
```

### 更新部署

```bash
# 拉取最新代码
git pull

# 重新构建并启动
docker-compose up -d --build
```

## 🔒 安全建议

1. **修改默认密码**：确保 `.env` 文件中的密码和密钥都已修改
2. **防火墙配置**：只开放必要的端口（80, 443, 8081）
3. **数据库安全**：PostgreSQL 端口（5432）建议仅内网访问
4. **HTTPS**：生产环境建议使用 Nginx 反向代理配置 HTTPS
5. **定期备份**：定期备份数据库数据

## 🌐 配置域名和 HTTPS

### 使用 Nginx 反向代理（推荐）

1. 安装 Nginx：

```bash
apt-get install nginx  # Ubuntu
# 或
yum install nginx      # CentOS
```

2. 配置 Nginx：

创建 `/etc/nginx/sites-available/fluent-life`：

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # 前端
    location / {
        proxy_pass http://localhost:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 后端 API
    location /api {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # WebSocket
    location /ws {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

3. 启用配置：

```bash
ln -s /etc/nginx/sites-available/fluent-life /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

4. 配置 HTTPS（使用 Let's Encrypt）：

```bash
apt-get install certbot python3-certbot-nginx
certbot --nginx -d your-domain.com
```

## 🐛 故障排查

### 服务无法启动

1. 检查日志：`docker-compose logs`
2. 检查端口占用：`netstat -tulpn | grep -E '80|8081|5432'`
3. 检查 Docker 状态：`systemctl status docker`

### 数据库连接失败

1. 检查数据库容器状态：`docker-compose ps postgres`
2. 检查环境变量：`docker-compose exec backend env | grep DB`
3. 查看数据库日志：`docker-compose logs postgres`

### 前端无法访问后端

1. 检查 API 地址配置：确认 `.env` 中的 `VITE_API_BASE_URL` 正确
2. 检查后端服务：`curl http://localhost:8081/health`
3. 检查 CORS 配置：查看后端日志

## 📞 支持

如遇问题，请查看：
- 项目日志：`docker-compose logs`
- 系统日志：`journalctl -u docker`
- GitHub Issues（如果有）

## 🔄 更新部署

```bash
# 1. 拉取最新代码
git pull

# 2. 停止服务
docker-compose down

# 3. 重新构建
docker-compose build --no-cache

# 4. 启动服务
docker-compose up -d

# 5. 查看日志确认
docker-compose logs -f
```

---

**注意**：首次部署后，请务必修改所有默认密码和密钥！


