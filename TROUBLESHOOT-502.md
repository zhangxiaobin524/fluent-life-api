# 502 Bad Gateway 错误排查指南

## 问题描述
访问 `http://120.55.250.184:8081/api/v1/training/records?page_size=100` 返回 502 错误。

## 502 错误常见原因
1. 后端服务没有运行或崩溃
2. Nginx 无法连接到后端服务
3. 网络配置问题
4. 后端服务启动失败
5. 端口冲突

## 排查步骤

### 1. 检查容器状态
```bash
# 查看所有容器状态
docker compose ps

# 或者
docker ps -a

# 应该看到三个容器都在运行：
# - fluent-life-db (PostgreSQL)
# - fluent-life-api (后端)
# - fluent-life-frontend (前端 Nginx)
```

**如果后端容器没有运行：**
```bash
# 查看后端容器日志
docker compose logs backend

# 或者
docker logs fluent-life-api

# 查看最近的错误
docker compose logs backend --tail 100
```

### 2. 检查后端服务是否正常
```bash
# 直接访问后端服务（绕过 Nginx）
curl http://localhost:8081/health

# 或者
curl http://120.55.250.184:8081/health

# 如果返回 200，说明后端服务正常
# 如果连接失败，说明后端服务有问题
```

### 3. 检查后端容器日志
```bash
# 查看实时日志
docker compose logs -f backend

# 查看最近的错误
docker compose logs backend --tail 50

# 常见错误：
# - 数据库连接失败
# - 端口被占用
# - 配置文件错误
# - 依赖服务未启动
```

### 4. 检查网络连接
```bash
# 从前端容器测试连接到后端
docker exec fluent-life-frontend wget -O- http://backend:8081/health

# 如果失败，说明网络配置有问题
```

### 5. 检查 Nginx 配置
```bash
# 查看前端容器中的 Nginx 配置
docker exec fluent-life-frontend cat /etc/nginx/conf.d/default.conf

# 检查 proxy_pass 配置是否正确
# 应该是：proxy_pass http://backend:8081;
```

### 6. 检查端口占用
```bash
# 检查 8081 端口是否被占用
netstat -tulpn | grep 8081

# 或者
ss -tulpn | grep 8081

# 如果被其他进程占用，需要停止或修改端口
```

### 7. 检查数据库连接
```bash
# 检查数据库容器是否运行
docker compose ps postgres

# 检查数据库连接
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"

# 查看数据库日志
docker compose logs postgres
```

## 常见问题及解决方案

### 问题1：后端容器启动失败
**症状：** `docker compose ps` 显示后端容器状态为 `Exited` 或 `Restarting`

**解决方案：**
```bash
# 查看详细错误
docker compose logs backend

# 常见原因：
# 1. 数据库连接失败 - 检查 DB_HOST, DB_PASSWORD 等环境变量
# 2. 配置文件错误 - 检查 configs/config.yaml
# 3. 端口被占用 - 检查端口占用情况

# 重启服务
docker compose restart backend
```

### 问题2：Nginx 无法连接到后端
**症状：** Nginx 日志显示 `connect() failed (111: Connection refused)`

**解决方案：**
```bash
# 检查后端服务是否在同一网络
docker network inspect fluent-life-api_fluent-life-network

# 检查后端容器名称
docker ps --filter "name=fluent-life-api"

# 确保 nginx.conf 中的 proxy_pass 使用正确的服务名
# 应该是：http://backend:8081
```

### 问题3：数据库连接失败
**症状：** 后端日志显示数据库连接错误

**解决方案：**
```bash
# 检查 .env 文件中的数据库配置
cat .env | grep DB_

# 确保配置正确：
# DB_HOST=postgres
# DB_PORT=5432
# DB_USER=fluent_life
# DB_PASSWORD=your_password
# DB_NAME=fluent_life

# 测试数据库连接
docker exec fluent-life-api wget -O- http://localhost:8081/health
```

### 问题4：端口冲突
**症状：** 容器无法绑定端口 8081

**解决方案：**
```bash
# 检查端口占用
lsof -i :8081
# 或
netstat -tulpn | grep 8081

# 如果被占用，停止占用进程或修改 docker-compose.yml 中的端口映射
```

## 快速修复命令

### 完全重启所有服务
```bash
cd /opt/fluent-life/fluent-life-api

# 停止所有服务
docker compose down

# 清理（可选）
docker compose down -v

# 重新构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f
```

### 只重启后端服务
```bash
# 停止后端
docker compose stop backend

# 重新构建后端（如果需要）
docker compose build backend

# 启动后端
docker compose up -d backend

# 查看日志
docker compose logs -f backend
```

### 检查服务健康状态
```bash
# 检查所有服务
docker compose ps

# 检查后端健康
curl http://localhost:8081/health

# 检查前端
curl http://localhost

# 检查数据库
docker exec fluent-life-db pg_isready -U fluent_life
```

## 调试技巧

### 进入容器调试
```bash
# 进入后端容器
docker exec -it fluent-life-api sh

# 检查进程
ps aux

# 检查端口监听
netstat -tulpn

# 检查环境变量
env | grep DB_
env | grep PORT
```

### 查看 Nginx 错误日志
```bash
# 查看 Nginx 错误日志
docker exec fluent-life-frontend cat /var/log/nginx/error.log

# 或者实时查看
docker compose logs -f frontend
```

### 测试 API 端点
```bash
# 直接测试后端 API（绕过 Nginx）
curl -v http://localhost:8081/api/v1/training/records?page_size=100

# 通过 Nginx 测试
curl -v http://localhost/api/v1/training/records?page_size=100

# 带认证（如果需要）
curl -v -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost/api/v1/training/records?page_size=100
```

## 预防措施

1. **定期检查日志**
   ```bash
   docker compose logs --tail 100
   ```

2. **设置健康检查**
   - docker-compose.yml 中已配置健康检查
   - 定期检查服务状态

3. **监控资源使用**
   ```bash
   docker stats
   ```

4. **备份配置**
   - 定期备份 .env 文件
   - 备份 docker-compose.yml

## 如果问题仍然存在

1. 收集以下信息：
   - `docker compose ps` 输出
   - `docker compose logs backend` 输出
   - `docker compose logs frontend` 输出
   - `curl http://localhost:8081/health` 输出
   - `.env` 文件内容（隐藏敏感信息）

2. 检查系统资源：
   ```bash
   # 内存使用
   free -h
   
   # 磁盘空间
   df -h
   
   # CPU 使用
   top
   ```

3. 检查 Docker 状态：
   ```bash
   docker info
   docker system df
   ```


