# Supabase 数据库设置指南

## 连接信息

- **Host**: `db.btmolnyjfnsaadsfcguc.supabase.co`
- **Port**: `6543` (Pooler) 或 `5432` (Direct)
- **User**: `postgres`
- **Password**: `cy!f.GPByAvE.6&`
- **Database**: `postgres`
- **SSL Mode**: `require`

## 连接方式说明

### 1. Supabase Pooler (推荐 - 端口 6543)
- 使用 PgBouncer 连接池
- 适合生产环境和高并发
- 更稳定，自动管理连接

### 2. Direct Connection (端口 5432)
- 直接连接数据库
- 适合开发环境
- 需要配置 IP 白名单

## 配置 IP 白名单

1. 登录 Supabase Dashboard
2. 进入 **Settings** > **Database**
3. 找到 **Connection Pooling** 或 **Network Restrictions**
4. 添加你的 IP 地址到白名单
5. 或者设置为允许所有 IP（仅开发环境）

## 创建数据库表

### 方法1: 使用脚本（推荐）

```bash
cd backend
./setup_supabase.sh
```

### 方法2: 手动执行 SQL

```bash
# 使用 Pooler 连接
psql "postgresql://postgres:cy%21f.GPByAvE.6%26@db.btmolnyjfnsaadsfcguc.supabase.co:6543/postgres?sslmode=require" -f migrations/create_tables.sql

# 或使用 Direct 连接（如果 Pooler 不可用）
psql "postgresql://postgres:cy%21f.GPByAvE.6%26@db.btmolnyjfnsaadsfcguc.supabase.co:5432/postgres?sslmode=require" -f migrations/create_tables.sql
```

### 方法3: 在 Supabase Dashboard 中执行

1. 登录 Supabase Dashboard
2. 进入 **SQL Editor**
3. 复制 `migrations/create_tables.sql` 的内容
4. 粘贴并执行

## 环境变量配置

如果不想在配置文件中写密码，可以使用环境变量：

```bash
export DB_HOST=db.btmolnyjfnsaadsfcguc.supabase.co
export DB_PORT=6543
export DB_USER=postgres
export DB_PASSWORD='cy!f.GPByAvE.6&'
export DB_NAME=postgres
export DB_SSLMODE=require
```

## 测试连接

```bash
# 测试 Pooler 连接
psql "postgresql://postgres:cy%21f.GPByAvE.6%26@db.btmolnyjfnsaadsfcguc.supabase.co:6543/postgres?sslmode=require" -c "SELECT version();"

# 测试 Direct 连接
psql "postgresql://postgres:cy%21f.GPByAvE.6%26@db.btmolnyjfnsaadsfcguc.supabase.co:5432/postgres?sslmode=require" -c "SELECT version();"
```

## 注意事项

1. **密码 URL 编码**：密码中的特殊字符需要编码
   - `!` → `%21`
   - `&` → `%26`

2. **SSL 必需**：Supabase 要求使用 SSL 连接

3. **连接限制**：Supabase 免费版有连接数限制，使用 Pooler 可以更好地管理连接

4. **IP 白名单**：如果连接超时，检查 Supabase Dashboard 中的 IP 白名单设置

## 故障排查

### 连接超时
- 检查 Supabase 项目是否运行中
- 检查 IP 是否在白名单中
- 尝试使用 Direct 连接（端口 5432）

### 认证失败
- 确认用户名是 `postgres`（不是你的 Supabase 账号名）
- 确认密码正确
- 检查密码是否需要 URL 编码

### SSL 错误
- 确保 `sslmode=require`
- 检查网络是否允许 SSL 连接







