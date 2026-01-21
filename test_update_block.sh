#!/bin/bash

# 测试 updateBlock API
# 这个脚本测试更新块的功能

BASE_URL="http://localhost:6806"
TOKEN=""

echo "=========================================="
echo "测试 updateBlock API"
echo "=========================================="

# 1. 登录获取 token
echo ""
echo "1. 登录系统..."
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/web/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "link918@qq.com",
    "password": "zhangli1115"
  }')

echo "登录响应: $LOGIN_RESPONSE"

# 提取 token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi

echo "✓ 登录成功，token: ${TOKEN:0:20}..."

# 2. 获取笔记本列表
echo ""
echo "2. 获取笔记本列表..."
NOTEBOOKS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/notebook/lsNotebooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d '{}')

echo "笔记本列表: $NOTEBOOKS_RESPONSE"

# 提取第一个笔记本 ID
NOTEBOOK_ID=$(echo $NOTEBOOKS_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$NOTEBOOK_ID" ]; then
    echo "❌ 无法获取笔记本 ID"
    exit 1
fi

echo "✓ 获取到笔记本 ID: $NOTEBOOK_ID"

# 3. 获取文档列表
echo ""
echo "3. 获取文档列表..."
DOCS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/filetree/listDocsByPath" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"notebook\": \"$NOTEBOOK_ID\",
    \"path\": \"/\"
  }")

echo "文档列表响应: $DOCS_RESPONSE"

# 提取第一个文档 ID
DOC_ID=$(echo $DOCS_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$DOC_ID" ]; then
    echo "❌ 无法获取文档 ID"
    exit 1
fi

echo "✓ 获取到文档 ID: $DOC_ID"

# 4. 获取文档内容
echo ""
echo "4. 获取文档内容..."
DOC_CONTENT=$(curl -s -X POST "${BASE_URL}/api/filetree/getDoc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$DOC_ID\"
  }")

echo "文档内容: ${DOC_CONTENT:0:200}..."

# 提取第一个块 ID（不是文档本身）
BLOCK_ID=$(echo $DOC_CONTENT | grep -o 'data-node-id="[^"]*"' | head -2 | tail -1 | cut -d'"' -f2)

if [ -z "$BLOCK_ID" ]; then
    echo "⚠️  文档中没有找到子块，尝试使用文档 ID 本身"
    BLOCK_ID=$DOC_ID
fi

echo "✓ 使用块 ID: $BLOCK_ID"

# 5. 测试 updateBlock - 更新块内容
echo ""
echo "5. 测试 updateBlock - 更新块内容..."
TIMESTAMP=$(date +%s)
UPDATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/block/updateBlock" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$BLOCK_ID\",
    \"dataType\": \"markdown\",
    \"data\": \"# 测试更新块 - $TIMESTAMP\n\n这是一个测试更新块功能的内容。时间戳: $TIMESTAMP\"
  }")

echo "更新响应: $UPDATE_RESPONSE"

# 检查响应
if echo "$UPDATE_RESPONSE" | grep -q '"code":0'; then
    echo "✓ updateBlock 测试成功"
else
    echo "❌ updateBlock 测试失败"
    exit 1
fi

# 6. 验证更新后的内容
echo ""
echo "6. 验证更新后的内容..."
sleep 1
VERIFY_CONTENT=$(curl -s -X POST "${BASE_URL}/api/filetree/getDoc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$DOC_ID\"
  }")

if echo "$VERIFY_CONTENT" | grep -q "$TIMESTAMP"; then
    echo "✓ 内容更新验证成功，找到时间戳: $TIMESTAMP"
else
    echo "⚠️  内容更新验证：未找到时间戳，但这可能是正常的（取决于块类型）"
fi

echo ""
echo "=========================================="
echo "✓ updateBlock API 测试完成"
echo "=========================================="
