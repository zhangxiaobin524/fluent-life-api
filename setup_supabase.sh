#!/bin/bash

# Supabase 数据库设置脚本

# 数据库连接信息
DB_HOST="db.btmolnyjfnsaadsfcguc.supabase.co"
DB_PORT="6543"
DB_USER="postgres"
DB_PASSWORD="cy!f.GPByAvE.6&"
DB_NAME="postgres"

# URL encode 密码中的特殊字符
# ! -> %21, & -> %26
ENCODED_PASSWORD="cy%21f.GPByAvE.6%26"

echo "🚀 连接到 Supabase 数据库..."

# 使用 Pooler 连接（端口 6543）
CONNECTION_STRING="postgresql://${DB_USER}:${ENCODED_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require"

echo "📊 测试连接..."
psql "$CONNECTION_STRING" -c "SELECT version();" 2>&1

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ 连接成功！"
    echo ""
    echo "📦 创建数据库表..."
    psql "$CONNECTION_STRING" -f migrations/create_tables.sql
    
    if [ $? -eq 0 ]; then
        echo ""
        echo "✅ 表创建成功！"
        echo ""
        echo "📋 验证表结构..."
        psql "$CONNECTION_STRING" -c "\dt"
    else
        echo "❌ 表创建失败，请检查错误信息"
        exit 1
    fi
else
    echo "❌ 连接失败，请检查："
    echo "   1. Supabase 项目是否运行中"
    echo "   2. IP 地址是否在白名单中（Supabase Dashboard > Settings > Database）"
    echo "   3. 密码是否正确"
    exit 1
fi







