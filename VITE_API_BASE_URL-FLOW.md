# VITE_API_BASE_URL 传递流程

## 传递路径

```
.env 文件
  ↓
docker-compose.yml (build.args)
  ↓
Dockerfile (ARG)
  ↓
Dockerfile (ENV)
  ↓
npm run build (Vite 构建时使用)
  ↓
前端代码中的 import.meta.env.VITE_API_BASE_URL
```

## 详细说明

### 1. `.env` 文件（源）
位置：`fluent-life-api/.env`

```bash
VITE_API_BASE_URL=http://120.55.250.184:8081/api/v1
```

### 2. `docker-compose.yml`（传递构建参数）
位置：`fluent-life-api/docker-compose.yml`

```yaml
frontend:
  build:
    context: ../fluent-life-frontend
    dockerfile: Dockerfile
    args:
      VITE_API_BASE_URL: ${VITE_API_BASE_URL:-http://localhost:8081/api/v1}
      # ↑ 从 .env 文件读取，如果不存在则使用默认值
```

### 3. `Dockerfile`（接收构建参数）
位置：`fluent-life-frontend/Dockerfile`

```dockerfile
ARG VITE_API_BASE_URL
# ↑ 接收从 docker-compose.yml 传递过来的构建参数

ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
# ↑ 设置为环境变量，供 npm run build 时使用
```

### 4. Vite 构建时使用
在 `npm run build` 时，Vite 会读取 `VITE_API_BASE_URL` 环境变量，并将其注入到前端代码中。

### 5. 前端代码中使用
位置：`fluent-life-frontend/services/api.ts`

```typescript
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8081/api/v1';
// ↑ 从构建时注入的环境变量读取
```

## 如何修改

### 方法1：修改 .env 文件（推荐）
```bash
cd /opt/fluent-life/fluent-life-api
vi .env

# 修改这一行
VITE_API_BASE_URL=http://your-actual-ip:8081/api/v1
```

### 方法2：直接传递构建参数
```bash
docker build \
  --build-arg VITE_API_BASE_URL=http://your-ip:8081/api/v1 \
  -t fluent-life-frontend \
  -f fluent-life-frontend/Dockerfile \
  fluent-life-frontend
```

## 验证

构建后，可以在前端代码中检查：
```bash
# 查看构建后的代码（在容器中）
docker exec fluent-life-frontend cat /usr/share/nginx/html/assets/*.js | grep -o 'VITE_API_BASE_URL[^"]*' | head -1
```

## 重要提示

1. **必须在构建时传递**：`VITE_API_BASE_URL` 是构建时变量，必须在 `docker build` 时传递
2. **运行时无法修改**：构建完成后，这个值就固定在前端代码中了
3. **必须重新构建**：修改 `.env` 后，必须重新构建前端镜像才能生效
4. **使用 --no-cache**：确保使用新的环境变量，建议使用 `--no-cache` 重新构建


