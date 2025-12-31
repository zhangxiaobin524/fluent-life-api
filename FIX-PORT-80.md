# 修复 80 端口外部无法访问问题

## 问题描述
- ✅ 服务器本地可以访问：`curl http://localhost:80` 成功
- ❌ 外部无法访问：`curl http://120.55.250.184` 失败

## 可能的原因

### 1. 防火墙没有开放 80 端口
### 2. 阿里云安全组没有开放 80 端口
### 3. Docker 端口映射问题
### 4. Nginx 服务没有正常运行

## 排查步骤

### 步骤1：检查防火墙（在服务器上执行）

```bash
# 检查防火墙状态
systemctl status firewalld

# 检查已开放的端口
firewall-cmd --list-ports

# 如果 80 不在列表中，开放它
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --reload

# 验证
sudo firewall-cmd --list-ports | grep 80
```

### 步骤2：检查 Docker 端口映射

```bash
# 检查前端容器端口映射
docker ps | grep fluent-life-frontend

# 应该看到：
# 0.0.0.0:80->80/tcp
# 如果是 127.0.0.1:80->80/tcp，需要修改 docker-compose.yml
```

### 步骤3：检查服务状态

```bash
# 检查容器是否运行
docker compose ps

# 检查前端容器日志
docker compose logs frontend --tail 20

# 检查端口监听
netstat -tulpn | grep :80
# 或
ss -tulpn | grep :80
```

### 步骤4：检查阿里云安全组

在阿里云控制台：

1. **进入 ECS 实例** → **安全组** → **配置规则**
2. **添加入方向规则**（如果还没有）：
   - **端口范围**：`80/80`
   - **协议类型**：`TCP`
   - **授权对象**：`0.0.0.0/0`（允许所有IP）
   - **描述**：HTTP端口
   - **策略**：允许

3. **确认 ECS 实例绑定了该安全组**

### 步骤5：测试内网IP访问

```bash
# 在服务器上测试内网IP
curl http://$(hostname -I | awk '{print $1}')

# 如果这个也失败，说明服务只监听 localhost
```

## 快速修复

### 修复防火墙（CentOS/RHEL）

```bash
# 开放 80 端口
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload

# 验证
sudo firewall-cmd --list-ports
```

### 修复防火墙（Ubuntu/Debian）

```bash
# 开放 80 端口
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8081/tcp

# 检查状态
sudo ufw status
```

## 一键诊断脚本

在服务器上执行：

```bash
#!/bin/bash

echo "=== 1. 检查端口监听 ==="
netstat -tulpn | grep :80
echo ""

echo "=== 2. 检查 Docker 容器 ==="
docker ps --format "table {{.Names}}\t{{.Ports}}" | grep fluent-life-frontend
echo ""

echo "=== 3. 检查防火墙 ==="
if systemctl is-active --quiet firewalld; then
    echo "防火墙：运行中"
    echo "已开放端口："
    firewall-cmd --list-ports
    echo ""
    if firewall-cmd --list-ports | grep -q "80"; then
        echo "✅ 80 端口已开放"
    else
        echo "❌ 80 端口未开放"
        echo "执行: sudo firewall-cmd --permanent --add-port=80/tcp && sudo firewall-cmd --reload"
    fi
else
    echo "防火墙：未运行"
fi
echo ""

echo "=== 4. 测试连接 ==="
echo "本地连接："
curl -s -o /dev/null -w "HTTP状态码: %{http_code}\n" http://localhost || echo "❌ 失败"
echo "内网IP连接："
curl -s -o /dev/null -w "HTTP状态码: %{http_code}\n" http://$(hostname -I | awk '{print $1}') || echo "❌ 失败"
echo ""

echo "=== 5. 服务器IP信息 ==="
echo "内网IP: $(hostname -I | awk '{print $1}')"
echo "公网IP: $(curl -s ifconfig.me 2>/dev/null || echo '无法获取')"
echo ""

echo "=== 6. 检查安全组 ==="
echo "请在阿里云控制台检查："
echo "1. ECS 实例 → 安全组 → 配置规则"
echo "2. 确认有入方向规则：端口 80/80，协议 TCP，授权对象 0.0.0.0/0"
echo "3. 确认 ECS 实例绑定了该安全组"
```

## 常见问题

### Q: 安全组规则添加了还是不生效？
A: 检查：
1. **端口范围**：必须是 `80/80`，不是 `80` 或 `80-80`
2. **授权对象**：必须是 `0.0.0.0/0`，不是 `127.0.0.1` 或你的IP
3. **规则方向**：必须是"入方向"，不是"出方向"
4. **ECS 实例绑定**：确认实例绑定了该安全组

### Q: 防火墙已开放，但还是连不上？
A: 检查：
1. 安全组是否配置
2. Docker 端口映射是否正确
3. 服务是否正常运行

### Q: 如何确认是防火墙还是安全组的问题？
A: 
1. 临时关闭防火墙测试：`sudo systemctl stop firewalld`
2. 如果关闭防火墙后可以访问，说明是防火墙问题
3. 如果还是不行，说明是安全组问题

## 推荐配置

### 阿里云安全组规则（必须配置）

**入方向规则**：
1. **HTTP (80)**：
   - 端口范围：`80/80`
   - 协议类型：`TCP`
   - 授权对象：`0.0.0.0/0`
   - 描述：HTTP端口

2. **HTTPS (443)**（如果使用）：
   - 端口范围：`443/443`
   - 协议类型：`TCP`
   - 授权对象：`0.0.0.0/0`
   - 描述：HTTPS端口

3. **后端API (8081)**（可选，建议仅内网）：
   - 端口范围：`8081/8081`
   - 协议类型：`TCP`
   - 授权对象：`0.0.0.0/0`（或指定IP范围）
   - 描述：后端API端口

4. **SSH (22)**：
   - 端口范围：`22/22`
   - 协议类型：`TCP`
   - 授权对象：`你的IP/32`（建议只允许你的IP）
   - 描述：SSH端口

### 服务器防火墙规则

```bash
# CentOS/RHEL
sudo firewall-cmd --permanent --add-port=80/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
```

## 验证修复

修复后，从外部测试：

```bash
# 从你的本地电脑测试
curl http://120.55.250.184

# 应该返回 HTML 内容（前端页面）
# 或使用浏览器访问
# http://120.55.250.184
```

## 如果还是不行

1. **检查安全组规则优先级**：
   - 如果有拒绝规则，优先级可能高于允许规则
   - 检查是否有其他规则阻止了 80 端口

2. **检查 ECS 实例网络类型**：
   - 确认是公网IP，不是内网IP
   - 检查是否开启了公网访问

3. **检查 Docker 网络**：
   ```bash
   docker network inspect fluent-life-api_fluent-life-network
   ```

4. **查看详细错误**：
   ```bash
   # 在服务器上查看 Nginx 日志
   docker compose logs frontend --tail 50
   ```



