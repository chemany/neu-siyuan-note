#!/bin/bash
# 思源笔记未引用资源定时清理脚本
# 每天凌晨0点执行，清理所有用户的未引用资源
# 
# 使用方法：
# 1. 手动执行: bash /root/code/siyuan/scripts/cleanup-unused-assets.sh
# 2. 定时执行: 添加到 crontab (0 0 * * * /root/code/siyuan/scripts/cleanup-unused-assets.sh)

LOG_FILE="/var/log/siyuan-cleanup.log"
SIYUAN_API="http://127.0.0.1:6806"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "========== 开始清理未引用资源 =========="

# 调用内部清理 API（仅允许本地访问，无需认证）
log "正在清理未引用资源..."
response=$(curl -s -X POST "$SIYUAN_API/api/system/cleanupUnusedAssets" \
    -H "Content-Type: application/json" \
    -d '{}')

# 检查 API 响应
if echo "$response" | grep -q '"code":0'; then
    # 提取删除的文件路径
    paths=$(echo "$response" | grep -o '"paths":\[[^]]*\]' | sed 's/"paths":\[//;s/\]$//')
    if [ -n "$paths" ] && [ "$paths" != "" ]; then
        count=$(echo "$paths" | tr ',' '\n' | wc -l)
        log "成功清理 $count 个未引用资源"
    else
        log "没有需要清理的资源"
    fi
else
    log "清理失败: $response"
    exit 1
fi

log "========== 清理完成 =========="
