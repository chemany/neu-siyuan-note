#!/bin/bash

echo "========================================="
echo "思源笔记多用户系统测试"
echo "========================================="

BASE_URL="http://localhost:6806"

echo ""
echo "1. 注册用户1..."
USER1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser1",
    "email": "user1@test.com",
    "password": "password123"
  }')

echo "响应: $USER1_RESPONSE"
USER1_TOKEN=$(echo $USER1_RESPONSE | grep -oP '"token":"[^"]*' | cut -d'"' -f4)
USER1_ID=$(echo $USER1_RESPONSE | grep -oP '"id":"[^"]*' | cut -d'"' -f4)

echo "用户1 Token: $USER1_TOKEN"
echo "用户1 ID: $USER1_ID"

echo ""
echo "2. 注册用户2..."
USER2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser2",
    "email": "user2@test.com",
    "password": "password456"
  }')

echo "响应: $USER2_RESPONSE"
USER2_TOKEN=$(echo $USER2_RESPONSE | grep -oP '"token":"[^"]*' | cut -d'"' -f4)
USER2_ID=$(echo $USER2_RESPONSE | grep -oP '"id":"[^"]*' | cut -d'"' -f4)

echo "用户2 Token: $USER2_TOKEN"
echo "用户2 ID: $USER2_ID"

echo ""
echo "3. 用户1登录..."
LOGIN1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user1@test.com",
    "password": "password123"
  }')

echo "响应: $LOGIN1_RESPONSE"

echo ""
echo "4. 获取用户1信息..."
PROFILE1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/profile" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER1_TOKEN")

echo "响应: $PROFILE1_RESPONSE"

echo ""
echo "5. 获取用户2信息..."
PROFILE2_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/profile" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $USER2_TOKEN")

echo "响应: $PROFILE2_RESPONSE"

echo ""
echo "6. 验证令牌1..."
VERIFY1_RESPONSE=$(curl -s -X POST "$BASE_URL/api/web/auth/verify-token" \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$USER1_TOKEN\"}")

echo "响应: $VERIFY1_RESPONSE"

echo ""
echo "7. 检查workspace目录..."
echo "主workspace: /home/jason/code/siyuan/workspace"
ls -la /home/jason/code/siyuan/workspace/
echo ""
echo "用户数据目录: /home/jason/code/siyuan/workspace/data/users"
if [ -f /home/jason/code/siyuan/workspace/data/users/users.json ]; then
  echo "users.json 内容:"
  cat /home/jason/code/siyuan/workspace/data/users/users.json | python3 -m json.tool
fi

echo ""
echo "========================================="
echo "测试完成!"
echo "========================================="
