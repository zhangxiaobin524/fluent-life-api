# 🚀 ECS 部署快速指南

## 一、服务器准备

### 1. 连接服务器
```bash
ssh root@your-ecs-ip
```

### 2. 安装 Docker 和 Docker Compose

**Ubuntu/Debian:**
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
systemctl start docker && systemctl enable docker
```

**CentOS/Alibaba Cloud Linux:**
```bash
yum install -y docker
systemctl start docker && systemctl enable docker
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
```

### 3. 开放端口
```bash
# 阿里云 ECS 需要在安全组中开放以下端口：
# - 80 (HTTP)
# - 8081 (后端 API)
# - 443 (HTTPS，如果使用)
```

## 二、部署应用

### 方法一：使用快速部署脚本（推荐）

```bash
# 1. 上传项目到服务器
# 可以使用 git clone 或 scp 上传

# 2. 进入项目目录
cd /opt/fluent-life  # 或你的项目目录

# 3. 配置环境变量
cp env.example .env
nano .env  # 编辑配置文件，修改密码和密钥

# 4. 执行快速部署
chmod +x quick-deploy.sh
./quick-deploy.sh
```

### 方法二：手动部署

```bash
# 1. 配置环境变量
cp env.example .env
nano .env

# 2. 构建镜像
docker-compose build

# 3. 启动服务
docker-compose up -d

# 4. 查看日志
docker-compose logs -f
```

## 三、环境变量配置

编辑 `.env` 文件，**必须修改**以下配置：

```env
# 数据库密码（必须修改）
DB_PASSWORD=your_secure_password_here

# JWT 密钥（必须修改，至少32个字符）
JWT_SECRET=your-secret-key-change-in-production-min-32-chars

# 前端 API 地址
# 如果使用域名：
VITE_API_BASE_URL=http://your-domain.com/api/v1
# 如果使用 IP：
VITE_API_BASE_URL=http://your-server-ip:8081/api/v1
```

## 四、验证部署

```bash
# 检查服务状态
docker-compose ps

# 检查后端健康
curl http://localhost:8081/health

# 检查前端
curl http://localhost

# 查看日志
docker-compose logs -f
```

## 五、常用命令

```bash
# 查看日志
docker-compose logs -f

# 重启服务
docker-compose restart

# 停止服务
docker-compose down

# 更新部署
git pull
docker-compose up -d --build
```

## 六、配置域名（可选）

### 使用 Nginx 反向代理

1. 安装 Nginx:
```bash
apt-get install nginx  # Ubuntu
# 或
yum install nginx      # CentOS
```

2. 配置 Nginx:
```bash
nano /etc/nginx/sites-available/fluent-life
```

添加配置：
```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:80;
        proxy_set_header Host $host;
    }

    location /api {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
    }

    location /ws {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

3. 启用配置:
```bash
ln -s /etc/nginx/sites-available/fluent-life /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

4. 配置 HTTPS:
```bash
apt-get install certbot python3-certbot-nginx
certbot --nginx -d your-domain.com
```

## 七、故障排查

### 服务无法启动
```bash
# 查看详细日志
docker-compose logs

# 检查端口占用
netstat -tulpn | grep -E '80|8081|5432'
```

### 数据库连接失败
```bash
# 检查数据库容器
docker-compose ps postgres
docker-compose logs postgres
```

### 前端无法访问后端
```bash
# 检查后端服务
curl http://localhost:8081/health

# 检查环境变量
docker-compose exec backend env | grep DB
```

## 八、安全建议

1. ✅ 修改所有默认密码和密钥
2. ✅ 只开放必要端口（80, 443）
3. ✅ 数据库端口（5432）仅内网访问
4. ✅ 使用 HTTPS（Let's Encrypt 免费证书）
5. ✅ 定期备份数据库

## 九、备份和恢复

### 备份数据库
```bash
docker-compose exec postgres pg_dump -U fluent_life fluent_life > backup.sql
```

### 恢复数据库
```bash
docker-compose exec -T postgres psql -U fluent_life fluent_life < backup.sql
```

---

**详细文档请查看**: `DEPLOY.md`


