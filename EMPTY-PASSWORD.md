# PostgreSQL 空密码配置

## .env 文件中空密码的写法

### 方法1：等号后留空（推荐）
```bash
DB_PASSWORD=
```

### 方法2：空字符串
```bash
DB_PASSWORD=""
```

### 方法3：单引号空字符串
```bash
DB_PASSWORD=''
```

## ⚠️ 重要提示

**PostgreSQL 默认不允许空密码！**

如果密码为空，PostgreSQL 会拒绝连接。有几种解决方案：

### 方案1：设置一个最小密码（推荐）

即使是最简单的密码也比空密码安全：

```bash
# .env 文件
DB_PASSWORD=123456
```

然后重置数据库用户密码：
```bash
docker exec fluent-life-db psql -U postgres -c "ALTER USER fluent_life WITH PASSWORD '123456';"
```

### 方案2：使用 PostgreSQL 的 trust 认证（仅开发环境）

如果确实需要空密码，可以修改 PostgreSQL 的认证配置，但这**非常不安全**，只适用于本地开发：

1. 修改 PostgreSQL 的 `pg_hba.conf`：
```bash
# 进入容器
docker exec -it fluent-life-db sh

# 编辑 pg_hba.conf（需要 root 权限）
# 将认证方式改为 trust
```

2. 或者使用环境变量：
```yaml
postgres:
  environment:
    POSTGRES_HOST_AUTH_METHOD: trust  # 允许无密码连接
```

**⚠️ 警告：生产环境绝对不要使用 trust 认证！**

### 方案3：使用环境变量覆盖（如果后端支持）

如果后端代码支持从环境变量读取空密码，可以：

```bash
# .env 文件
DB_PASSWORD=

# 后端代码需要处理空字符串的情况
```

## 推荐配置

### 开发环境
```bash
# .env
DB_PASSWORD=123456
```

### 生产环境
```bash
# .env
DB_PASSWORD=your_secure_password_here
# 使用强密码，至少 12 位，包含大小写字母、数字和特殊字符
```

## 当前配置检查

检查你的配置：

```bash
# 1. 查看 .env 文件
cat /opt/fluent-life/fluent-life-api/.env | grep DB_PASSWORD

# 2. 查看后端环境变量
docker exec fluent-life-api env | grep DB_PASSWORD

# 3. 测试数据库连接
docker exec fluent-life-db psql -U fluent_life -d fluent_life -c "SELECT 1;"
```

## 如果必须使用空密码

如果后端代码确实需要空密码（不推荐），可以：

1. 在 `.env` 文件中：
   ```bash
   DB_PASSWORD=
   ```

2. 修改 PostgreSQL 认证方式（仅开发环境）：
   ```yaml
   postgres:
     environment:
       POSTGRES_HOST_AUTH_METHOD: trust
   ```

3. 重新创建容器：
   ```bash
   docker compose stop postgres
   docker compose rm -f postgres
   docker compose up -d postgres
   ```

## 最佳实践

**强烈建议设置一个密码，即使是最简单的：**

```bash
# .env 文件
DB_PASSWORD=123456

# 然后重置数据库用户密码
docker exec fluent-life-db psql -U postgres -c "ALTER USER fluent_life WITH PASSWORD '123456';"

# 重启后端
docker compose restart backend
```


