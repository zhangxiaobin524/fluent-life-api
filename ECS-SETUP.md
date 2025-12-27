# 🚀 CentOS ECS 服务器完整部署指南

## 📋 第一步：服务器初始化（全新服务器必做）

### 方法一：使用初始化脚本（推荐）

1. **将初始化脚本上传到服务器**：

```bash
# 在本地执行
scp init-centos-server.sh root@your-server-ip:/root/
```

2. **SSH 连接到服务器并执行**：

```bash
ssh root@your-server-ip
bash init-centos-server.sh
```

脚本会自动安装：
- ✅ Docker
- ✅ Docker Compose
- ✅ Git、curl、wget 等基础工具
- ✅ 配置防火墙
- ✅ 优化系统配置
- ✅ 创建项目目录

### 方法二：手动安装

如果脚本执行失败，可以手动执行以下命令：

```bash
# 1. 更新系统
yum update -y
yum install -y epel-release

# 2. 安装基础工具
yum install -y curl wget git vim net-tools

# 3. 安装 Docker
yum install -y yum-utils
yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
yum install -y docker-ce docker-ce-cli containerd.io
systemctl start docker
systemctl enable docker

# 4. 安装 Docker Compose
curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose

# 5. 配置防火墙
firewall-cmd --permanent --add-port=80/tcp
firewall-cmd --permanent --add-port=443/tcp
firewall-cmd --permanent --add-port=8081/tcp
firewall-cmd --reload
```

## 📦 第二步：上传项目到服务器

### 方法一：使用上传脚本（推荐）

在**本地项目目录**执行：

```bash
./upload-to-server.sh root@your-server-ip
```

### 方法二：使用 Git（如果项目在 Git 仓库）

```bash
ssh root@your-server-ip
cd /opt/fluent-life
git clone https://your-repo-url.git .
```

### 方法三：使用 scp 手动上传

```bash
# 在本地项目目录执行
scp -r . root@your-server-ip:/opt/fluent-life/
```

## ⚙️ 第三步：配置环境变量

```bash
ssh root@your-server-ip
cd /opt/fluent-life

# 复制环境变量示例文件
cp env.example .env

# 编辑配置文件
nano .env
```

**必须修改的配置项**：

```env
# 数据库密码（必须修改为强密码）
DB_PASSWORD=your_secure_password_here

# JWT 密钥（必须修改，至少32个字符）
JWT_SECRET=your-secret-key-change-in-production-min-32-chars

# 前端 API 地址
# 如果使用 IP 访问：
VITE_API_BASE_URL=http://your-server-ip:8081/api/v1
# 如果使用域名：
VITE_API_BASE_URL=http://your-domain.com/api/v1
```

保存文件：`Ctrl + O`，然后 `Enter`，最后 `Ctrl + X`

## 🚀 第四步：部署应用

```bash
cd /opt/fluent-life

# 赋予脚本执行权限
chmod +x quick-deploy.sh

# 执行快速部署
./quick-deploy.sh
```

部署过程包括：
1. 构建 Docker 镜像
2. 启动所有服务（数据库、后端、前端）
3. 检查服务状态

## ✅ 第五步：验证部署

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

## 🌐 访问应用

部署成功后，可以通过以下地址访问：

- **前端**: `http://your-server-ip`
- **后端 API**: `http://your-server-ip:8081`
- **健康检查**: `http://your-server-ip:8081/health`

## 📝 常用命令

```bash
# 查看所有服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
docker-compose logs -f backend    # 只看后端日志
docker-compose logs -f frontend   # 只看前端日志
docker-compose logs -f postgres   # 只看数据库日志

# 重启服务
docker-compose restart
docker-compose restart backend    # 重启特定服务

# 停止服务
docker-compose down

# 更新部署（代码更新后）
cd /opt/fluent-life
git pull  # 或重新上传代码
docker-compose up -d --build
```

## 🔒 安全配置

### 1. 修改 SSH 端口（可选但推荐）

```bash
nano /etc/ssh/sshd_config
# 修改 Port 22 为其他端口，如 2222
systemctl restart sshd
```

### 2. 配置 ECS 安全组

在阿里云/腾讯云控制台，确保安全组开放以下端口：
- **80** (HTTP)
- **443** (HTTPS，如果使用)
- **8081** (后端 API，可选，建议仅内网访问)
- **22** (SSH)

### 3. 配置域名和 HTTPS（推荐）

参考 `DEPLOY.md` 中的域名配置部分。

## 🐛 故障排查

### 问题1: Docker 无法启动

```bash
# 检查 Docker 状态
systemctl status docker

# 查看 Docker 日志
journalctl -u docker

# 重启 Docker
systemctl restart docker
```

### 问题2: 端口被占用

```bash
# 查看端口占用
netstat -tulpn | grep -E '80|8081|5432'

# 停止占用端口的服务
systemctl stop nginx  # 如果 Nginx 占用了 80 端口
```

### 问题3: 数据库连接失败

```bash
# 检查数据库容器
docker-compose ps postgres
docker-compose logs postgres

# 检查环境变量
docker-compose exec backend env | grep DB
```

### 问题4: 前端无法访问后端

```bash
# 检查后端服务
curl http://localhost:8081/health

# 检查前端构建时的环境变量
docker-compose exec frontend env | grep VITE
```

### 问题5: 内存不足

```bash
# 查看内存使用
free -h

# 清理 Docker 资源
docker system prune -a
```

## 📊 监控和维护

### 查看资源使用

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
df -h

# 查看系统负载
htop
```

### 备份数据库

```bash
# 创建备份
docker-compose exec postgres pg_dump -U fluent_life fluent_life > backup_$(date +%Y%m%d).sql

# 恢复备份
docker-compose exec -T postgres psql -U fluent_life fluent_life < backup_20240101.sql
```

## 🔄 更新应用

```bash
cd /opt/fluent-life

# 1. 备份数据库（重要！）
docker-compose exec postgres pg_dump -U fluent_life fluent_life > backup_before_update.sql

# 2. 拉取最新代码
git pull
# 或重新上传代码文件

# 3. 停止服务
docker-compose down

# 4. 重新构建并启动
docker-compose up -d --build

# 5. 检查服务状态
docker-compose ps
docker-compose logs -f
```

## 📞 获取帮助

如果遇到问题：

1. 查看日志：`docker-compose logs`
2. 检查服务状态：`docker-compose ps`
3. 查看系统资源：`htop`、`df -h`、`free -h`
4. 检查网络连接：`ping`、`curl`

---

**提示**: 首次部署后，请务必修改所有默认密码和密钥！


