# 修复防火墙端口访问问题

## 问题描述
无法从外部访问 `http://120.55.250.184:8081`，连接超时。

## 可能的原因
1. 防火墙没有开放 8081 端口
2. 安全组没有配置 8081 端口
3. 服务没有正常运行

## 排查步骤

### 步骤1：检查服务是否运行
```bash
# 在服务器上执行
docker compose ps

# 检查后端服务是否在运行
curl http://localhost:8081/health
```

### 步骤2：检查防火墙状态（CentOS/RHEL）
```bash
# 检查防火墙状态
systemctl status firewalld

# 或者
firewall-cmd --state
```

### 步骤3：检查已开放的端口
```bash
# 查看所有开放的端口
firewall-cmd --list-ports

# 查看所有开放的服务
firewall-cmd --list-services
```

## 解决方案

### 方案1：开放 8081 端口（CentOS/RHEL 使用 firewalld）

```bash
# 1. 开放 8081 端口
sudo firewall-cmd --permanent --add-port=8081/tcp

# 2. 重新加载防火墙规则
sudo firewall-cmd --reload

# 3. 验证端口是否开放
sudo firewall-cmd --list-ports | grep 8081
```

### 方案2：开放 8081 端口（Ubuntu/Debian 使用 ufw）

```bash
# 1. 开放 8081 端口
sudo ufw allow 8081/tcp

# 2. 检查状态
sudo ufw status
```

### 方案3：开放 8081 端口（使用 iptables）

```bash
# 1. 开放 8081 端口
sudo iptables -A INPUT -p tcp --dport 8081 -j ACCEPT

# 2. 保存规则（CentOS/RHEL）
sudo service iptables save

# 或者（Ubuntu/Debian）
sudo iptables-save > /etc/iptables/rules.v4
```

### 方案4：临时关闭防火墙（仅用于测试，不推荐）

```bash
# CentOS/RHEL
sudo systemctl stop firewalld

# Ubuntu/Debian
sudo ufw disable
```

## ECS 安全组配置

如果使用阿里云/腾讯云/华为云等 ECS，还需要在云控制台配置安全组：

### 阿里云 ECS
1. 登录阿里云控制台
2. 进入 ECS 实例
3. 点击"安全组" → "配置规则"
4. 添加规则：
   - 端口范围：8081/8081
   - 协议类型：TCP
   - 授权对象：0.0.0.0/0（允许所有IP，或指定IP）
   - 描述：后端API端口

### 腾讯云 CVM
1. 登录腾讯云控制台
2. 进入 CVM 实例
3. 点击"安全组" → "修改规则"
4. 添加规则：
   - 类型：自定义
   - 协议端口：TCP:8081
   - 来源：0.0.0.0/0
   - 策略：允许

### 华为云 ECS
1. 登录华为云控制台
2. 进入 ECS 实例
3. 点击"安全组" → "入方向规则" → "添加规则"
4. 配置：
   - 协议端口：TCP:8081
   - 源地址：0.0.0.0/0
   - 描述：后端API端口

## 一键修复脚本（CentOS/RHEL）

```bash
#!/bin/bash

# 检查防火墙状态
if systemctl is-active --quiet firewalld; then
    echo "防火墙正在运行，开放 8081 端口..."
    
    # 开放端口
    sudo firewall-cmd --permanent --add-port=8081/tcp
    sudo firewall-cmd --permanent --add-port=80/tcp
    sudo firewall-cmd --permanent --add-port=443/tcp
    
    # 重新加载
    sudo firewall-cmd --reload
    
    # 验证
    echo "已开放的端口："
    sudo firewall-cmd --list-ports
    
    echo "✅ 防火墙配置完成"
else
    echo "防火墙未运行，跳过配置"
fi

# 检查服务状态
echo "检查服务状态..."
docker compose ps

# 测试本地连接
echo "测试本地连接..."
curl http://localhost:8081/health || echo "❌ 本地连接失败，检查服务是否运行"
```

## 验证修复

### 在服务器上测试
```bash
# 1. 测试本地连接
curl http://localhost:8081/health

# 2. 检查端口监听
netstat -tulpn | grep 8081
# 或
ss -tulpn | grep 8081

# 3. 检查防火墙规则
sudo firewall-cmd --list-ports
```

### 从外部测试
```bash
# 从你的本地电脑测试
curl http://120.55.250.184:8081/health

# 或者使用 telnet 测试端口是否开放
telnet 120.55.250.184 8081
```

## 常见问题

### Q: 防火墙已开放，但还是连不上？
A: 检查：
1. ECS 安全组是否配置
2. 服务是否正常运行：`docker compose ps`
3. 端口是否正确映射：`docker compose ps` 查看端口映射

### Q: 如何查看服务器IP？
```bash
# 查看公网IP
curl ifconfig.me

# 查看所有IP
ip addr show
```

### Q: 如何临时关闭防火墙测试？
```bash
# CentOS/RHEL
sudo systemctl stop firewalld

# 测试后记得重新启动
sudo systemctl start firewalld
```

## 推荐配置

生产环境建议只开放必要的端口：

```bash
# 开放常用端口
sudo firewall-cmd --permanent --add-port=80/tcp      # HTTP
sudo firewall-cmd --permanent --add-port=443/tcp     # HTTPS
sudo firewall-cmd --permanent --add-port=8081/tcp    # 后端API
sudo firewall-cmd --reload
```


