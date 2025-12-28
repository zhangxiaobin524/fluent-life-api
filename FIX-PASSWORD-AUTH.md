# 修复 PostgreSQL 密码认证失败问题

## 问题描述
- 可以直接用 `psql` 连接成功
- 但后端服务连接时报 `password authentication failed`

## 原因分析
密码不一致：
- 数据库中 `fluent_life` 用户的密码 ≠ `.env` 文件中的 `DB_PASSWORD`
- 或者后端服务没有正确读取环境变量

## 解决方案

### 方案1：重置数据库用户密码（推荐）

将数据库中 `fluent_life` 用户的密码改为与 `.env` 文件中的 `DB_PASSWORD` 一致：

```bash
# 1. 查看 .env 文件中的密码
cat /opt/fluent-life/fluent-life-api/.env | grep DB_PASSWORD

# 2. 使用 postgres 超级用户重置 fluent_life 用户的密码
# 假设 .env 中的密码是 123456
docker exec fluent-life-db psql -U postgres -c "ALTER USER fluent_life WITH PASSWORD '123456';"

# 3. 验证密码是否正确
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"
```

### 方案2：修改 .env 文件中的密码

如果数据库中的密码是另一个值，修改 `.env` 文件：

```bash
# 1. 编辑 .env 文件
vi /opt/fluent-life/fluent-life-api/.env

# 2. 修改 DB_PASSWORD 为数据库中的实际密码
# DB_PASSWORD=actual_password_here

# 3. 重启后端服务
docker compose restart backend
```

### 方案3：检查环境变量是否正确传递

```bash
# 1. 检查后端容器中的环境变量
docker exec fluent-life-api env | grep DB_

# 2. 应该看到：
# DB_USER=fluent_life
# DB_PASSWORD=123456 (或你的实际密码)
# DB_HOST=postgres
# DB_NAME=fluent_life

# 3. 如果环境变量不对，检查 docker-compose.yml 配置
```

## 快速修复步骤

### 步骤1：确认 .env 文件中的密码
```bash
cd /opt/fluent-life/fluent-life-api
cat .env | grep DB_PASSWORD
```

### 步骤2：重置数据库用户密码
```bash
# 假设密码是 123456，如果不是请替换
docker exec fluent-life-db psql -U postgres -c "ALTER USER fluent_life WITH PASSWORD '123456';"
```

### 步骤3：验证密码
```bash
# 使用新密码测试连接
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"
```

### 步骤4：重启后端服务
```bash
docker compose restart backend

# 查看日志确认连接成功
docker compose logs backend --tail 30 | grep -i database
```

## 如果还是不行

### 检查后端环境变量
```bash
# 查看后端容器的环境变量
docker exec fluent-life-api env | grep DB_

# 确认：
# - DB_USER=fluent_life
# - DB_PASSWORD 与 .env 文件中的一致
# - DB_HOST=postgres
# - DB_NAME=fluent_life
```

### 检查数据库连接字符串
后端可能使用连接字符串，检查配置：

```bash
# 查看后端日志中的连接信息（注意不要暴露完整密码）
docker compose logs backend | grep -i "database\|connect" | tail -10
```

### 完全重置（如果数据不重要）

如果数据可以重新创建：

```bash
# 停止所有服务
docker compose down

# 删除数据库卷（⚠️ 会丢失所有数据）
docker volume rm fluent-life-api_postgres_data

# 重新启动（会使用 .env 中的密码创建新数据库）
docker compose up -d
```

## 验证修复

```bash
# 1. 测试数据库连接
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"

# 2. 检查后端日志
docker compose logs backend --tail 20

# 3. 测试 API
curl http://localhost:8081/health
```


