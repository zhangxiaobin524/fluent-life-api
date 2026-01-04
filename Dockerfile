# 多阶段构建：构建阶段
FROM golang:1.24 AS builder

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apt-get update && \
    apt-get install -y git ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN GOPROXY=https://goproxy.cn,direct go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fluent-life-api ./cmd/server/main.go

# 运行阶段
FROM ubuntu:22.04

# 设置时区避免交互式安装
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Shanghai

# 安装必要的运行时依赖
RUN apt-get update && \
    apt-get install -y ca-certificates tzdata wget && \
    rm -rf /var/lib/apt/lists/* && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/fluent-life-api .
COPY --from=builder /app/configs ./configs

# 创建非 root 用户
RUN groupadd -g 1000 appuser && \
    useradd -m -u 1000 -g appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# 暴露端口
EXPOSE 8081

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# 启动应用
CMD ["./fluent-life-api"]





