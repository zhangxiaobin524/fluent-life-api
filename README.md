# Fluent Life - API 后端项目

流畅生活（Fluent Life）后端 API 服务。

## 技术栈

- Go 1.21+
- Gin Web Framework
- PostgreSQL / Supabase
- JWT 认证

## 开发

### 环境要求

- Go 1.21 或更高版本
- PostgreSQL 数据库（或使用 Supabase）

### 安装依赖

```bash
go mod download
```

### 配置

复制并编辑配置文件：

```bash
cp configs/config.yaml.example configs/config.yaml
```

编辑 `configs/config.yaml` 设置数据库连接等信息。

### 运行

```bash
# 开发模式
go run cmd/server/main.go

# 或编译后运行
go build -o fluent-life-backend cmd/server/main.go
./fluent-life-backend
```

服务将在 `http://localhost:8080` 启动。

## API 文档

### 认证相关

- `POST /api/v1/auth/send-code` - 发送验证码
- `POST /api/v1/auth/register` - 注册
- `POST /api/v1/auth/login` - 登录
- `POST /api/v1/auth/refresh` - 刷新 Token

### 用户相关

- `GET /api/v1/users/profile` - 获取用户资料
- `PUT /api/v1/users/profile` - 更新用户资料
- `GET /api/v1/users/stats` - 获取统计数据

### 训练记录

- `POST /api/v1/training/records` - 创建训练记录
- `GET /api/v1/training/records` - 获取训练记录
- `GET /api/v1/training/stats` - 获取训练统计
- `GET /api/v1/training/meditation-progress` - 获取冥想进度

### 社区

- `GET /api/v1/community/posts` - 获取帖子列表
- `POST /api/v1/community/posts` - 创建帖子
- `POST /api/v1/community/posts/:id/like` - 点赞/取消点赞
- `GET /api/v1/community/posts/:id/comments` - 获取评论
- `POST /api/v1/community/posts/:id/comments` - 创建评论

### AI 导师

- `POST /api/v1/ai/chat` - AI 对话
- `GET /api/v1/ai/conversation` - 获取对话历史
- `POST /api/v1/ai/analyze-speech` - 分析语音

### 成就系统

- `GET /api/v1/achievements` - 获取成就列表

## 健康检查

```bash
curl http://localhost:8080/health
```

## 数据库迁移

数据库表结构在 `migrations/create_tables.sql` 中定义。

## CORS 配置

后端已配置 CORS，允许前端域名访问。如需修改，请编辑 `internal/middleware/cors.go`。
