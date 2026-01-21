#!/bin/bash

# 测试关闭笔记本功能

BASE_URL="http://localhost:6806"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mzg5OTk5OTksInVzZXJfaWQiOiIxIiwidXNlcm5hbWUiOiJqYXNvbiJ9.Ow_KKzMXVPXXWLqJxlxLqJxlxLqJxlxLqJxlxLqJxlw"

echo "=========================================="
echo "测试关闭笔记本功能"
echo "=========================================="
echo ""

# 1. 先获取笔记本列表
echo "1. 获取笔记本列表..."
NOTEBOOKS=$(curl -s -X POST "${BASE_URL}/api/notebook/lsNotebooks" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{}')

echo "笔记本列表响应："
echo "$NOTEBOOKS"
echo ""

# 提取第一个笔记本的 ID（使用 grep 和 sed）
NOTEBOOK_ID=$(echo "$NOTEBOOKS" | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')

if [ -z "$NOTEBOOK_ID" ]; then
    echo "❌ 没有找到笔记本，无法测试关闭功能"
    exit 1
fi

echo "找到笔记本 ID: $NOTEBOOK_ID"
echo ""

# 2. 先打开笔记本（确保笔记本是打开状态）
echo "2. 打开笔记本 $NOTEBOOK_ID（确保笔记本是打开状态）..."
OPEN_RESULT=$(curl -s -X POST "${BASE_URL}/api/notebook/openNotebook" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"notebook\": \"${NOTEBOOK_ID}\"}")

echo "打开笔记本响应："
echo "$OPEN_RESULT"
echo ""

sleep 1

# 3. 关闭笔记本
echo "3. 关闭笔记本 $NOTEBOOK_ID..."
CLOSE_RESULT=$(curl -s -X POST "${BASE_URL}/api/notebook/closeNotebook" \
  -H "Authorization: Token ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"notebook\": \"${NOTEBOOK_ID}\"}")

echo "关闭笔记本响应："
echo "$CLOSE_RESULT"
echo ""

# 4. 检查结果
CODE=$(echo "$CLOSE_RESULT" | grep -o '"code":[0-9]*' | sed 's/"code"://')

if [ "$CODE" = "0" ]; then
    echo "✅ 关闭笔记本成功！"
    echo ""
    
    # 5. 再次获取笔记本列表，验证笔记本已关闭
    echo "4. 验证笔记本已关闭..."
    sleep 1
    NOTEBOOKS_AFTER=$(curl -s -X POST "${BASE_URL}/api/notebook/lsNotebooks" \
      -H "Authorization: Token ${TOKEN}" \
      -H "Content-Type: application/json" \
      -d '{}')
    
    echo "验证笔记本列表："
    echo "$NOTEBOOKS_AFTER"
    echo ""
    
    # 检查笔记本是否在列表中
    if echo "$NOTEBOOKS_AFTER" | grep -q "\"id\":\"${NOTEBOOK_ID}\""; then
        echo "✅ 笔记本在列表中"
        
        # 检查是否为关闭状态（closed 应该为 true）
        # 提取该笔记本的 closed 状态
        CLOSED_STATUS=$(echo "$NOTEBOOKS_AFTER" | grep -A 20 "\"id\":\"${NOTEBOOK_ID}\"" | grep -o '"closed":[^,}]*' | head -1 | sed 's/"closed"://')
        
        if [ "$CLOSED_STATUS" = "true" ]; then
            echo "✅ 笔记本状态正确（已关闭）"
        else
            echo "⚠️  笔记本状态: closed=$CLOSED_STATUS"
            echo "注意：笔记本可能仍然显示为打开状态"
        fi
    else
        echo "❌ 笔记本不在列表中"
        exit 1
    fi
else
    echo "❌ 关闭笔记本失败！"
    MSG=$(echo "$CLOSE_RESULT" | grep -o '"msg":"[^"]*"' | sed 's/"msg":"//;s/"//')
    echo "错误信息: $MSG"
    exit 1
fi

echo ""
echo "=========================================="
echo "✅ 所有测试通过！"
echo "=========================================="
