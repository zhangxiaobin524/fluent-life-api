# 只重启 PostgreSQL 容器（不重启其他服务）

## 方法1：只重启 PostgreSQL 容器（推荐）

```bash
cd /opt/fluent-life/fluent-life-api

# 只停止 PostgreSQL 容器
docker compose stop postgres

# 只启动 PostgreSQL 容器
docker compose start postgres

# 或者一条命令
docker compose restart postgres
```

## 方法2：重新创建 PostgreSQL 容器（如果修改了环境变量）

如果修改了 `docker-compose.yml` 中的 PostgreSQL 配置（如用户名、密码），需要重新创建容器：

```bash
cd /opt/fluent-life/fluent-life-api

# 只停止并删除 PostgreSQL 容器（保留数据卷）
docker compose stop postgres
docker compose rm -f postgres

# 只启动 PostgreSQL 容器（使用新配置）
docker compose up -d postgres
```

## 方法3：修改配置后只重启相关容器

如果修改了 PostgreSQL 配置，可能还需要重启后端服务（因为后端需要重新连接数据库）：

```bash
cd /opt/fluent-life/fluent-life-api

# 1. 重启 PostgreSQL
docker compose restart postgres

# 2. 等待 PostgreSQL 启动完成（可选）
sleep 5

# 3. 重启后端服务（让它重新连接数据库）
docker compose restart backend

# 前端不需要重启，因为它不直接连接数据库
```

## 方法4：使用 Docker 命令直接操作

```bash
# 只重启 PostgreSQL 容器
docker restart fluent-life-db

# 或者
docker stop fluent-life-db
docker start fluent-life-db
```

## 验证 PostgreSQL 是否正常

```bash
# 检查容器状态
docker compose ps postgres

# 或者
docker ps | grep fluent-life-db

# 检查日志
docker compose logs postgres --tail 20

# 测试连接
docker exec fluent-life-db psql -U zhangxiaobin -d fluent_life -c "SELECT 1;"
```

## 注意事项

1. **数据持久化**：PostgreSQL 数据存储在 Docker volume 中，重启容器不会丢失数据
2. **服务依赖**：如果后端服务正在运行，重启 PostgreSQL 可能会导致短暂的连接错误，但会自动重连
3. **健康检查**：后端服务的 `depends_on` 配置会等待 PostgreSQL 健康检查通过

## 如果修改了环境变量

如果修改了 `.env` 文件或 `docker-compose.yml` 中的 PostgreSQL 配置：

```bash
# 1. 停止 PostgreSQL
docker compose stop postgres

# 2. 删除容器（数据卷会保留）
docker compose rm -f postgres

# 3. 重新创建并启动（使用新配置）
docker compose up -d postgres

# 4. 检查日志确认启动成功
docker compose logs -f postgres
```

## 快速命令总结

```bash
# 只重启 PostgreSQL
docker compose restart postgres

# 只重启后端（如果需要）
docker compose restart backend

# 查看 PostgreSQL 日志
docker compose logs -f postgres

# 测试数据库连接
docker exec fluent-life-db psql -U zhangxiaobin -d fluent_life -c "\l"
```


