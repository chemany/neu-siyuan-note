#!/bin/bash

# 测试创建笔记本功能
# 用于验证 createNotebook API 的多用户架构重构

echo "=========================================="
echo "测试创建笔记本功能"
echo "=========================================="

# 配置
BASE_URL="http://localhost:6806"
EMAIL="link918@qq.com"
PASSWORD="zhangli1115"

# 1. 登录获取 token
echo ""
echo "1. 登录获取 token..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

echo "登录响应: $LOGIN_RESPONSE"

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi

echo "✓ 登录成功，token: ${TOKEN:0:20}..."

# 2. 创建笔记本
echo ""
echo "2. 创建笔记本..."
NOTEBOOK_NAME="测试笔记本_$(date +%Y%m%d_%H%M%S)"
echo "笔记本名称: $NOTEBOOK_NAME"

CREATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/notebook/createNotebook" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d "{\"name\":\"${NOTEBOOK_NAME}\"}")

echo "创建响应: $CREATE_RESPONSE"

# 检查响应
if echo "$CREATE_RESPONSE" | grep -q '"code":0'; then
    echo "✓ 创建笔记本成功"
    
    # 提取笔记本 ID
    NOTEBOOK_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "笔记本 ID: $NOTEBOOK_ID"
else
    echo "❌ 创建笔记本失败"
    exit 1
fi

# 3. 验证笔记本是否在列表中
echo ""
echo "3. 验证笔记本是否在列表中..."
LIST_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/notebook/lsNotebooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}")

echo "笔记本列表响应: $LIST_RESPONSE"

if echo "$LIST_RESPONSE" | grep -q "$NOTEBOOK_ID"; then
    echo "✓ 笔记本已在列表中"
else
    echo "❌ 笔记本不在列表中"
    exit 1
fi

# 4. 检查笔记本是否已打开
if echo "$LIST_RESPONSE" | grep -q "\"id\":\"$NOTEBOOK_ID\".*\"closed\":false"; then
    echo "✓ 笔记本已自动打开"
else
    echo "⚠ 笔记本未自动打开（可能需要手动打开）"
fi

echo ""
echo "=========================================="
echo "✓ 所有测试通过！"
echo "=========================================="
echo ""
echo "创建的笔记本信息："
echo "  名称: $NOTEBOOK_NAME"
echo "  ID: $NOTEBOOK_ID"
echo ""
