# 修复前端 API 地址配置

## 问题描述
部署后前端调用的 API 地址是 `http://your-domain.com/api/v1/...`，这是一个占位符，需要修改为实际的服务器地址。

## 修复步骤

### 1. 登录服务器
```bash
ssh root@your-server-ip
```

### 2. 找到项目目录
通常项目在 `/opt/fluent-life/fluent-life-api` 或你部署的目录

```bash
cd /opt/fluent-life/fluent-life-api
# 或者
cd /path/to/your/project/fluent-life-api
```

### 3. 检查并修改 .env 文件
```bash
# 查看当前的 .env 文件
cat .env

# 编辑 .env 文件
vi .env
# 或
nano .env
```

### 4. 修改 VITE_API_BASE_URL
找到这一行：
```bash
VITE_API_BASE_URL=http://your-domain.com/api/v1
```

修改为实际的服务器地址：

**如果使用 IP 地址（推荐）：**
```bash
VITE_API_BASE_URL=http://120.55.250.184:8081/api/v1
```

**如果使用域名：**
```bash
VITE_API_BASE_URL=http://your-actual-domain.com/api/v1
# 或使用 HTTPS
VITE_API_BASE_URL=https://your-actual-domain.com/api/v1
```

**如果使用 Nginx 反向代理（前端和后端在同一域名下）：**
```bash
VITE_API_BASE_URL=http://your-domain.com/api/v1
# 或
VITE_API_BASE_URL=https://your-domain.com/api/v1
```

### 5. 保存并退出编辑器
- 如果使用 `vi`：按 `Esc`，然后输入 `:wq` 保存退出
- 如果使用 `nano`：按 `Ctrl+X`，然后按 `Y` 确认，再按 `Enter`

### 6. 验证环境变量
```bash
# 检查环境变量是否正确
cat .env | grep VITE_API_BASE_URL
```

### 7. 重新构建前端镜像
```bash
# 停止服务
docker compose down

# 重新构建前端镜像（必须使用 --no-cache 确保使用新的环境变量）
docker compose build --no-cache frontend

# 启动服务
docker compose up -d
```

### 8. 验证修复
```bash
# 检查前端容器中的环境变量
docker exec fluent-life-frontend env | grep VITE

# 查看前端日志
docker compose logs frontend

# 访问前端页面，打开浏览器开发者工具（F12），查看 Network 标签页
# API 请求应该使用新的地址，而不是 your-domain.com
```

## 常见问题

### Q: 修改后还是显示 your-domain.com？
A: 确保：
1. `.env` 文件中的值已正确修改
2. 使用了 `--no-cache` 重新构建
3. 前端容器已重启

### Q: 如何查看服务器 IP 地址？
```bash
# 方法1：查看公网IP
curl ifconfig.me

# 方法2：查看所有IP
ip addr show

# 方法3：查看内网IP
hostname -I
```

### Q: 如果使用域名，需要配置什么？
1. 确保域名已解析到服务器 IP
2. 如果使用 HTTPS，需要配置 SSL 证书
3. 如果使用 Nginx 反向代理，需要配置 Nginx

## 快速修复命令（一键执行）

```bash
# 进入项目目录
cd /opt/fluent-life/fluent-life-api

# 获取服务器IP（自动替换）
SERVER_IP=$(curl -s ifconfig.me)
sed -i "s|VITE_API_BASE_URL=.*|VITE_API_BASE_URL=http://${SERVER_IP}:8081/api/v1|g" .env

# 验证修改
cat .env | grep VITE_API_BASE_URL

# 重新构建并启动
docker compose down
docker compose build --no-cache frontend
docker compose up -d

echo "✅ 修复完成！API 地址已更新为: http://${SERVER_IP}:8081/api/v1"
```



