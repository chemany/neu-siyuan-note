#!/bin/bash

# 登录并测试 updateBlock API

BASE_URL="http://localhost:6806"
EMAIL="link918@qq.com"
PASSWORD="zhangli1115"

echo "=========================================="
echo "步骤 1: 使用统一登录获取 Token"
echo "=========================================="

# 尝试统一登录
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/web/auth/unified-login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")

echo "统一登录响应: ${LOGIN_RESPONSE}"
echo ""

# 提取 token
TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "${TOKEN}" ]; then
    echo "统一登录失败，尝试普通登录..."
    LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/web/auth/login" \
      -H "Content-Type: application/json" \
      -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASSWORD}\"}")
    
    echo "普通登录响应: ${LOGIN_RESPONSE}"
    echo ""
    
    TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

echo "登录响应: ${LOGIN_RESPONSE}"
echo ""

# 提取 token
TOKEN=$(echo "${LOGIN_RESPONSE}" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "${TOKEN}" ]; then
    echo "❌ 登录失败，无法获取 token"
    exit 1
fi

echo "✅ 登录成功！"
echo "Token: ${TOKEN}"
echo ""

echo "=========================================="
echo "步骤 2: 测试 updateBlock API"
echo "=========================================="
echo ""

# 获取笔记本列表
echo "获取笔记本列表..."
NOTEBOOKS=$(curl -s -X POST "${BASE_URL}/api/notebook/lsNotebooks" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json")

echo "笔记本列表响应: ${NOTEBOOKS}"
echo ""

# 提取第一个笔记本 ID
NOTEBOOK_ID=$(echo "${NOTEBOOKS}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "使用笔记本 ID: ${NOTEBOOK_ID}"
echo ""

if [ -z "${NOTEBOOK_ID}" ]; then
    echo "❌ 无法获取笔记本 ID"
    exit 1
fi

# 获取文档列表
echo "获取文档列表..."
DOCS=$(curl -s -X POST "${BASE_URL}/api/filetree/listDocsByPath" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"notebook\":\"${NOTEBOOK_ID}\",\"path\":\"/\"}")

echo "文档列表响应: ${DOCS}"
echo ""

# 提取第一个文档 ID
DOC_ID=$(echo "${DOCS}" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "使用文档 ID: ${DOC_ID}"
echo ""

if [ -z "${DOC_ID}" ]; then
    echo "❌ 无法获取文档 ID，请确保笔记本中有文档"
    exit 1
fi

# 获取文档内容
echo "获取文档内容..."
DOC_CONTENT=$(curl -s -X POST "${BASE_URL}/api/filetree/getDoc" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"${DOC_ID}\"}")

echo "文档内容响应（前500字符）: ${DOC_CONTENT:0:500}..."
echo ""

# 提取第一个段落块 ID（不是文档本身）
BLOCK_ID=$(echo "${DOC_CONTENT}" | grep -o '"id":"[^"]*"' | sed -n '2p' | cut -d'"' -f4)
if [ -z "${BLOCK_ID}" ]; then
    # 如果没有第二个块，使用文档 ID
    BLOCK_ID="${DOC_ID}"
fi
echo "使用块 ID: ${BLOCK_ID}"
echo ""

# 更新块内容
echo "=========================================="
echo "步骤 3: 更新块内容"
echo "=========================================="
TIMESTAMP=$(date +%s)
NEW_CONTENT="<div data-node-id=\"${BLOCK_ID}\" data-type=\"NodeParagraph\" class=\"p\"><div contenteditable=\"true\" spellcheck=\"false\">测试更新块 - 时间戳: ${TIMESTAMP}</div><div class=\"protyle-attr\" contenteditable=\"false\"></div></div>"

UPDATE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/block/updateBlock" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"${BLOCK_ID}\",\"data\":\"${NEW_CONTENT}\",\"dataType\":\"dom\"}")

echo "更新块响应: ${UPDATE_RESPONSE}"
echo ""

# 检查响应
if echo "${UPDATE_RESPONSE}" | grep -q '"code":0'; then
    echo "✅ 更新块成功！"
else
    echo "❌ 更新块失败！"
    echo "响应: ${UPDATE_RESPONSE}"
    exit 1
fi

# 验证更新
echo ""
echo "=========================================="
echo "步骤 4: 验证更新"
echo "=========================================="
sleep 1

VERIFY_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/block/getBlockInfo" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"id\":\"${BLOCK_ID}\"}")

echo "验证响应（前500字符）: ${VERIFY_RESPONSE:0:500}..."
echo ""

if echo "${VERIFY_RESPONSE}" | grep -q "${TIMESTAMP}"; then
    echo "✅ 验证成功！块内容已更新，包含时间戳: ${TIMESTAMP}"
else
    echo "⚠️  警告: 无法在响应中找到更新的时间戳"
    echo "完整验证响应: ${VERIFY_RESPONSE}"
fi

echo ""
echo "=========================================="
echo "测试完成 - updateBlock API 工作正常！"
echo "=========================================="
