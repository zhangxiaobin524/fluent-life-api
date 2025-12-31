# 修复阿里云 ECS 外部无法访问问题

## 问题描述
- ✅ 服务器本地可以访问：`curl http://localhost:8081/health` 成功
- ❌ 外部无法访问：`curl http://120.55.250.184:8081/health` 失败
- ✅ 已配置安全组规则

## 可能的原因

### 1. 后端服务只监听 localhost（最可能）
如果后端服务只监听 `127.0.0.1:8081` 而不是 `0.0.0.0:8081`，外部无法访问。

### 2. Docker 端口映射问题
检查端口是否正确映射到 `0.0.0.0`。

### 3. 安全组规则配置错误
检查安全组规则是否正确配置。

### 4. 防火墙还有规则阻止
检查是否有其他防火墙规则。

## 排查步骤

### 步骤1：检查服务监听地址

```bash
# 在服务器上检查端口监听情况
netstat -tulpn | grep 8081
# 或
ss -tulpn | grep 8081

# 查看输出，应该看到：
# tcp  0  0  0.0.0.0:8081  0.0.0.0:*  LISTEN  ...
# 如果是 127.0.0.1:8081，说明只监听本地，需要修复
```

### 步骤2：检查 Docker 端口映射

```bash
# 检查容器端口映射
docker ps | grep fluent-life-api

# 应该看到：
# 0.0.0.0:8081->8081/tcp

# 检查容器内部监听
docker exec fluent-life-api netstat -tulpn | grep 8081
```

### 步骤3：检查安全组规则

在阿里云控制台检查：
1. 进入 ECS 实例
2. 点击"安全组" → "配置规则"
3. 检查入方向规则：
   - 端口：8081/8081
   - 协议：TCP
   - 授权对象：0.0.0.0/0（允许所有IP）
   - 策略：允许

### 步骤4：检查防火墙

```bash
# 检查防火墙状态
systemctl status firewalld

# 检查已开放的端口
firewall-cmd --list-ports

# 如果 8081 不在列表中，开放它
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload
```

## 解决方案

### 方案1：修复后端服务监听地址（如果只监听 localhost）

检查后端代码中的监听地址：

```go
// 应该监听 0.0.0.0，而不是 127.0.0.1
addr := "0.0.0.0:" + cfg.Port
// 或
addr := ":" + cfg.Port  // 默认监听所有接口
```

### 方案2：检查 Docker 端口映射

确保 `docker-compose.yml` 中端口映射正确：

```yaml
backend:
  ports:
    - "8081:8081"  # 映射到 0.0.0.0:8081
    # 不要写成 "127.0.0.1:8081:8081"
```

### 方案3：使用 Nginx 反向代理（推荐）

如果后端只监听 localhost，可以通过 Nginx 反向代理：

前端 Nginx 已经配置了代理，但可能需要调整。

### 方案4：检查阿里云安全组规则

确保安全组规则配置正确：

1. **入方向规则**：
   - 端口范围：8081/8081
   - 协议类型：TCP
   - 授权对象：0.0.0.0/0
   - 描述：后端API端口

2. **检查规则是否生效**：
   - 规则添加后立即生效
   - 如果还是不行，尝试删除规则重新添加

## 快速诊断脚本

在服务器上执行：

```bash
#!/bin/bash

echo "=== 检查端口监听 ==="
netstat -tulpn | grep 8081
echo ""

echo "=== 检查 Docker 容器 ==="
docker ps | grep fluent-life-api
echo ""

echo "=== 检查容器内部监听 ==="
docker exec fluent-life-api netstat -tulpn 2>/dev/null | grep 8081 || echo "无法检查容器内部"
echo ""

echo "=== 检查防火墙 ==="
if systemctl is-active --quiet firewalld; then
    echo "防火墙状态：运行中"
    echo "已开放端口："
    firewall-cmd --list-ports
else
    echo "防火墙状态：未运行"
fi
echo ""

echo "=== 测试本地连接 ==="
curl -s http://localhost:8081/health && echo "✅ 本地连接成功" || echo "❌ 本地连接失败"
echo ""

echo "=== 检查服务器IP ==="
echo "内网IP: $(hostname -I | awk '{print $1}')"
echo "公网IP: $(curl -s ifconfig.me)"
```

## 常见问题

### Q: 安全组规则添加了还是不生效？
A: 检查：
1. 规则是否添加到正确的安全组
2. ECS 实例是否绑定了该安全组
3. 规则的方向是"入方向"不是"出方向"
4. 授权对象是 `0.0.0.0/0` 不是 `127.0.0.1`

### Q: 如何确认安全组规则生效？
A: 
1. 在阿里云控制台查看安全组规则
2. 确认 ECS 实例绑定了该安全组
3. 尝试从不同网络测试

### Q: 后端服务只监听 localhost 怎么办？
A: 
1. 修改代码监听 `0.0.0.0`
2. 或通过 Nginx 反向代理（前端 Nginx 已配置）

## 推荐配置

### 方案A：直接暴露后端端口（当前配置）

```yaml
backend:
  ports:
    - "8081:8081"  # 映射到所有接口
```

需要：
- ✅ 安全组开放 8081 端口
- ✅ 防火墙开放 8081 端口
- ✅ 后端监听 0.0.0.0:8081

### 方案B：只通过 Nginx 访问（更安全）

```yaml
backend:
  ports:
    - "127.0.0.1:8081:8081"  # 只监听本地
```

前端通过 Nginx 代理访问，外部只访问 80 端口。

## 验证步骤

1. **在服务器上测试**：
   ```bash
   curl http://localhost:8081/health
   curl http://127.0.0.1:8081/health
   curl http://$(hostname -I | awk '{print $1}'):8081/health
   ```

2. **从外部测试**：
   ```bash
   curl http://120.55.250.184:8081/health
   ```

3. **检查端口监听**：
   ```bash
   netstat -tulpn | grep 8081
   # 应该看到 0.0.0.0:8081，不是 127.0.0.1:8081
   ```



