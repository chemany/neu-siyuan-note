#!/bin/bash

# 思源笔记Web服务重启脚本

# 停止现有服务
echo "停止现有思源笔记服务..."
pkill -9 siyuan-kernel

# 等待进程完全退出
sleep 2

# 设置环境变量
export SIYUAN_WORKSPACE_PATH="/home/jason/code/siyuan/workspace"
export SIYUAN_WEB_MODE=true
export SIYUAN_JWT_SECRET="siyuan-web-auth-secret-change-in-production"
export SIYUAN_USER_DATA_ROOT="/mnt/nas-sata12/MindOcean/user-data/notes"

# 切换到kernel目录
cd /home/jason/code/siyuan/kernel

# 启动服务
echo "启动思源笔记Web服务..."
nohup ./siyuan-kernel --port 6806 > /tmp/siyuan.log 2>&1 &

# 等待服务启动
sleep 3

# 检查服务状态
if ps aux | grep -v grep | grep siyuan-kernel > /dev/null; then
    echo "✅ 思源笔记Web服务启动成功!"
    echo "访问地址: http://localhost:6806"
    echo "日志文件: /tmp/siyuan.log"
else
    echo "❌ 服务启动失败,请查看日志: /tmp/siyuan.log"
    exit 1
fi
