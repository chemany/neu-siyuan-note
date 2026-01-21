#!/bin/bash

# 测试删除块功能
# 这个脚本会：
# 1. 登录获取 token
# 2. 创建一个测试文档
# 3. 在文档中插入一个块
# 4. 删除这个块
# 5. 验证块已被删除

BASE_URL="http://localhost:6806"

echo "=========================================="
echo "测试删除块功能"
echo "=========================================="

# 1. 登录
echo ""
echo "1. 登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/system/login" \
  -H "Content-Type: application/json" \
  -d '{
    "userName": "jason",
    "userPassword": "zhangli1115",
    "captcha": ""
  }')

echo "登录响应: $LOGIN_RESPONSE"

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi
echo "✓ 登录成功，token: ${TOKEN:0:20}..."

# 2. 获取笔记本列表
echo ""
echo "2. 获取笔记本列表..."
NOTEBOOKS_RESPONSE=$(curl -s -X POST "$BASE_URL/api/notebook/lsNotebooks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d '{}')

echo "笔记本列表响应: $NOTEBOOKS_RESPONSE"

NOTEBOOK_ID=$(echo $NOTEBOOKS_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$NOTEBOOK_ID" ]; then
    echo "❌ 无法获取笔记本 ID"
    exit 1
fi
echo "✓ 获取到笔记本 ID: $NOTEBOOK_ID"

# 3. 创建测试文档
echo ""
echo "3. 创建测试文档..."
TIMESTAMP=$(date +%s)
DOC_PATH="/测试删除块_$TIMESTAMP"

CREATE_DOC_RESPONSE=$(curl -s -X POST "$BASE_URL/api/filetree/createDocWithMd" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"notebook\": \"$NOTEBOOK_ID\",
    \"path\": \"$DOC_PATH\",
    \"markdown\": \"# 测试删除块\\n\\n这是第一段。\\n\\n这是第二段，将被删除。\\n\\n这是第三段。\"
  }")

echo "创建文档响应: $CREATE_DOC_RESPONSE"

DOC_ID=$(echo $CREATE_DOC_RESPONSE | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$DOC_ID" ]; then
    echo "❌ 创建文档失败"
    exit 1
fi
echo "✓ 创建文档成功，文档 ID: $DOC_ID"

# 等待文档创建完成
sleep 1

# 4. 获取文档内容，找到要删除的块
echo ""
echo "4. 获取文档内容..."
GET_DOC_RESPONSE=$(curl -s -X POST "$BASE_URL/api/filetree/getDoc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$DOC_ID\",
    \"mode\": 0,
    \"size\": 102400
  }")

echo "文档内容响应（前500字符）: ${GET_DOC_RESPONSE:0:500}..."

# 从响应中提取第二个段落的 ID（包含"这是第二段"的块）
BLOCK_TO_DELETE=$(echo $GET_DOC_RESPONSE | grep -o 'data-node-id="[^"]*"[^>]*>这是第二段' | grep -o 'data-node-id="[^"]*"' | cut -d'"' -f2)

if [ -z "$BLOCK_TO_DELETE" ]; then
    echo "⚠️  无法找到要删除的块 ID，尝试手动查找..."
    # 尝试另一种方式提取
    echo "完整响应: $GET_DOC_RESPONSE" | head -c 1000
    echo ""
    echo "❌ 无法继续测试"
    exit 1
fi
echo "✓ 找到要删除的块 ID: $BLOCK_TO_DELETE"

# 5. 删除块
echo ""
echo "5. 删除块..."
DELETE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/block/deleteBlock" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$BLOCK_TO_DELETE\"
  }")

echo "删除块响应: $DELETE_RESPONSE"

# 检查响应是否成功
if echo "$DELETE_RESPONSE" | grep -q '"code":0'; then
    echo "✓ 删除块成功"
else
    echo "❌ 删除块失败"
    exit 1
fi

# 6. 验证块已被删除
echo ""
echo "6. 验证块已被删除..."
sleep 1

VERIFY_RESPONSE=$(curl -s -X POST "$BASE_URL/api/filetree/getDoc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"id\": \"$DOC_ID\",
    \"mode\": 0,
    \"size\": 102400
  }")

echo "验证响应（前500字符）: ${VERIFY_RESPONSE:0:500}..."

# 检查"这是第二段"是否还存在
if echo "$VERIFY_RESPONSE" | grep -q "这是第二段"; then
    echo "❌ 块未被删除，仍然存在于文档中"
    exit 1
else
    echo "✓ 块已成功删除"
fi

# 7. 清理：删除测试文档
echo ""
echo "7. 清理测试文档..."
REMOVE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/filetree/removeDoc" \
  -H "Content-Type: application/json" \
  -H "Authorization: Token $TOKEN" \
  -d "{
    \"notebook\": \"$NOTEBOOK_ID\",
    \"path\": \"$DOC_PATH\"
  }")

echo "删除文档响应: $REMOVE_RESPONSE"

echo ""
echo "=========================================="
echo "✓ 删除块功能测试通过！"
echo "=========================================="
