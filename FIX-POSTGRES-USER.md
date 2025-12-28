# 修复 PostgreSQL 用户不存在问题

## 问题描述
错误：`role "zhangxiaobin" does not exist`

这说明 PostgreSQL 数据库中不存在 `zhangxiaobin` 用户。

## 原因分析
配置不一致：
- `docker-compose.yml` 中配置的是 `POSTGRES_USER: zhangxiaobin`
- 但实际数据库中可能只有 `fluent_life` 用户（之前创建的）

## 解决方案

### 方案1：检查当前数据库中的用户（推荐先执行）

```bash
# 使用默认的 postgres 用户连接（这个用户总是存在的）
docker exec fluent-life-db psql -U postgres -d fluent_life -c "\du"

# 或者检查所有用户
docker exec fluent-life-db psql -U postgres -c "\du"
```

### 方案2：统一使用 fluent_life 用户（推荐）

如果数据库中已经有 `fluent_life` 用户，修改 `docker-compose.yml` 统一使用它：

```yaml
postgres:
  environment:
    POSTGRES_USER: fluent_life  # 改为 fluent_life
    POSTGRES_PASSWORD: ${DB_PASSWORD}
    POSTGRES_DB: fluent_life
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U fluent_life"]  # 已经是 fluent_life，不需要改
```

然后重新创建 PostgreSQL 容器：
```bash
docker compose stop postgres
docker compose rm -f postgres
docker compose up -d postgres
```

### 方案3：在 PostgreSQL 中创建 zhangxiaobin 用户

如果你想使用 `zhangxiaobin` 用户，需要在数据库中创建它：

```bash
# 使用 postgres 超级用户连接
docker exec -it fluent-life-db psql -U postgres

# 在 psql 中执行：
CREATE USER zhangxiaobin WITH PASSWORD '123456';
ALTER USER zhangxiaobin CREATEDB;
GRANT ALL PRIVILEGES ON DATABASE fluent_life TO zhangxiaobin;
\q
```

或者一条命令：
```bash
docker exec fluent-life-db psql -U postgres -c "CREATE USER zhangxiaobin WITH PASSWORD '123456';"
docker exec fluent-life-db psql -U postgres -c "ALTER USER zhangxiaobin CREATEDB;"
docker exec fluent-life-db psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE fluent_life TO zhangxiaobin;"
```

### 方案4：重新创建数据库容器（如果数据不重要）

如果数据可以重新创建，直接重新创建容器：

```bash
# 停止并删除容器和卷（⚠️ 会丢失数据）
docker compose stop postgres
docker compose rm -f postgres
docker volume rm fluent-life-api_postgres_data

# 重新创建
docker compose up -d postgres
```

## 快速修复步骤（推荐）

### 步骤1：检查当前用户
```bash
docker exec fluent-life-db psql -U postgres -c "\du"
```

### 步骤2：根据结果选择方案

**如果看到 `fluent_life` 用户存在：**
```bash
# 修改 docker-compose.yml，统一使用 fluent_life
# 然后重新创建容器
docker compose stop postgres
docker compose rm -f postgres
docker compose up -d postgres
```

**如果想创建 `zhangxiaobin` 用户：**
```bash
# 创建用户
docker exec fluent-life-db psql -U postgres -c "CREATE USER zhangxiaobin WITH PASSWORD '123456';"
docker exec fluent-life-db psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE fluent_life TO zhangxiaobin;"

# 测试连接
docker exec fluent-life-db psql -U zhangxiaobin -d fluent_life -c "SELECT 1;"
```

### 步骤3：更新后端配置

确保后端也使用相同的用户：

```yaml
backend:
  environment:
    DB_USER: fluent_life  # 或 zhangxiaobin，与 PostgreSQL 用户一致
    DB_PASSWORD: ${DB_PASSWORD}
```

### 步骤4：重启后端服务
```bash
docker compose restart backend
```

## 验证修复

```bash
# 测试数据库连接
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"

# 或
docker exec fluent-life-db psql -U zhangxiaobin -d fluent_life -c "SELECT 1;"

# 检查后端日志
docker compose logs backend --tail 20
```


