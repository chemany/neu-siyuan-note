#!/bin/bash

# 思源笔记笔记本优化功能测试脚本
# 用于测试按笔记本名称组织数据，增强AI分析功能

echo "开始测试思源笔记笔记本优化功能..."

# 设置测试环境
export SIYUAN_WORKSPACE_DIR="/root/code/MindOcean/user-data"
export SIYUAN_DATA_DIR="/root/code/MindOcean/user-data/notes"

# 测试API端点
BASE_URL="http://localhost:6806"
API_TOKEN="22rr68lsokt4tgti"

# 测试用户
USERNAME="jason"

echo "测试环境设置完成"
echo "工作空间: $SIYUAN_WORKSPACE_DIR"
echo "数据目录: $SIYUAN_DATA_DIR"
echo ""

# 函数：测试API调用
test_api() {
    local endpoint="$1"
    local data="$2"
    local description="$3"
    
    echo "测试: $description"
    echo "端点: $endpoint"
    echo "数据: $data"
    
    response=$(curl -s -X POST \
        -H "Authorization: Token $API_TOKEN" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "$BASE_URL$endpoint")
    
    echo "响应: $response"
    echo "---"
    
    # 检查响应是否包含成功标识
    if echo "$response" | grep -q '"success": true'; then
        echo "✅ $description - 成功"
        return 0
    else
        echo "❌ $description - 失败"
        return 1
    fi
}

# 1. 测试组织笔记本分类
echo "1. 测试组织笔记本分类..."
organize_data="{\"username\": \"$USERNAME\"}"
test_api "/api/notebook/organizeByCategory" "$organize_data" "组织笔记本分类"

# 2. 测试准备AI分析
echo ""
echo "2. 测试准备AI分析..."
prepare_data="{\"username\": \"$USERNAME\"}"
test_api "/api/notebook/prepareForAI" "$prepare_data" "准备AI分析"

# 3. 测试获取优化后的笔记本
echo ""
echo "3. 测试获取优化后的笔记本..."
get_data="{\"username\": \"$USERNAME\"}"
test_api "/api/notebook/getOptimized" "$get_data" "获取优化后的笔记本"

# 4. 测试搜索笔记本内容（如果存在笔记本）
echo ""
echo "4. 测试搜索笔记本内容..."
if [ -d "/root/code/MindOcean/user-data/notes/$USERNAME/organized" ]; then
    # 获取第一个笔记本名称
    first_notebook=$(ls -1 "/root/code/MindOcean/user-data/notes/$USERNAME/organized" | head -n 1)
    if [ -n "$first_notebook" ]; then
        search_data="{\"username\": \"$USERNAME\", \"notebook\": \"$first_notebook\", \"query\": \"文档\"}"
        test_api "/api/notebook/searchContent" "$search_data" "搜索笔记本内容"
    else
        echo "❌ 搜索笔记本内容 - 没有找到笔记本"
    fi
else
    echo "❌ 搜索笔记本内容 - 优化目录不存在"
fi

# 5. 检查文件结构
echo ""
echo "5. 检查文件结构..."
organized_path="/root/code/MindOcean/user-data/notes/$USERNAME/organized"

if [ -d "$organized_path" ]; then
    echo "✅ 组织目录存在: $organized_path"
    
    # 列出笔记本
    echo "笔记本列表:"
    for notebook in "$organized_path"/*; do
        if [ -d "$notebook" ]; then
            notebook_name=$(basename "$notebook")
            echo "  📁 $notebook_name"
            
            # 检查子目录
            for subdir in documents rich-notes vectors assets analysis; do
                subdir_path="$notebook/$subdir"
                if [ -d "$subdir_path" ]; then
                    echo "    ✓ $subdir/"
                else
                    echo "    ✗ $subdir/ (不存在)"
                fi
            done
            
            # 检查元数据文件
            metadata_file="$notebook/metadata.json"
            if [ -f "$metadata_file" ]; then
                echo "    ✓ metadata.json"
                # 显示统计信息
                if command -v jq >/dev/null 2>&1; then
                    echo "      统计: $(jq -r '.stats' "$metadata_file" 2>/dev/null || echo "解析失败")"
                fi
            else
                echo "    ✗ metadata.json (不存在)"
            fi
        fi
    done
    
    # 检查AI分析数据
    echo ""
    echo "AI分析数据:"
    for notebook in "$organized_path"/*; do
        if [ -d "$notebook" ]; then
            analysis_path="$notebook/analysis"
            index_file="$analysis_path/index.json"
            
            if [ -f "$index_file" ]; then
                notebook_name=$(basename "$notebook")
                echo "  📊 $notebook_name - analysis/index.json"
                
                if command -v jq >/dev/null 2>&1; then
                    docs_count=$(jq -r '.totalDocuments // 0' "$index_file" 2>/dev/null)
                    echo "    文档数量: $docs_count"
                fi
            fi
        fi
    done
    
else
    echo "❌ 组织目录不存在: $organized_path"
fi

# 6. 性能测试
echo ""
echo "6. 性能测试..."
if [ -d "$organized_path" ]; then
    start_time=$(date +%s.%N)
    
    # 统计总文件数
    total_files=0
    for notebook in "$organized_path"/*; do
        if [ -d "$notebook" ]; then
            # 使用find命令统计文件
            files_in_notebook=$(find "$notebook" -type f | wc -l)
            total_files=$((total_files + files_in_notebook))
        fi
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "N/A")
    
    echo "✅ 统计完成 - 总文件数: $total_files, 耗时: ${duration}秒"
    
    # 计算目录大小
    if command -v du >/dev/null 2>&1; then
        total_size=$(du -sh "$organized_path" 2>/dev/null | cut -f1)
        echo "✅ 目录大小: $total_size"
    fi
else
    echo "❌ 性能测试 - 组织目录不存在"
fi

# 7. 生成测试报告
echo ""
echo "7. 生成测试报告..."
report_file="/tmp/siyuan-notebook-optimization-report.txt"

cat > "$report_file" << EOF
思源笔记笔记本优化功能测试报告
生成时间: $(date)
测试用户: $USERNAME

测试结果总结:
1. 组织笔记本分类: $(curl -s -X POST -H "Authorization: Token $API_TOKEN" -H "Content-Type: application/json" -d "{\"username\": \"$USERNAME\"}" "$BASE_URL/api/notebook/organizeByCategory" | grep -o '"success":[^,]*' | cut -d: -f2 || echo "N/A")
2. 准备AI分析: $(curl -s -X POST -H "Authorization: Token $API_TOKEN" -H "Content-Type: application/json" -d "{\"username\": \"$USERNAME\"}" "$BASE_URL/api/notebook/prepareForAI" | grep -o '"success":[^,]*' | cut -d: -f2 || echo "N/A")
3. 获取优化笔记本: $(curl -s -X POST -H "Authorization: Token $API_TOKEN" -H "Content-Type: application/json" -d "{\"username\": \"$USERNAME\"}" "$BASE_URL/api/notebook/getOptimized" | grep -o '"success":[^,]*' | cut -d: -f2 || echo "N/A")

文件结构:
- 组织目录: $([ -d "$organized_path" ] && echo "存在" || echo "不存在")
- 笔记本数量: $([ -d "$organized_path" ] && ls -1 "$organized_path" 2>/dev/null | grep -c '^' || echo "0")

优化建议:
1. 确保思源笔记服务正常运行在端口 6806
2. 检查用户数据权限
3. 验证API token配置正确
4. 定期运行优化脚本以保持数据组织

测试完成时间: $(date)
EOF

echo "✅ 测试报告已生成: $report_file"
echo "报告内容预览:"
head -20 "$report_file"

echo ""
echo "🎉 思源笔记笔记本优化功能测试完成！"
echo ""
echo "使用说明:"
echo "1. 运行 ./siyuan/test-notebook-optimization.sh 进行测试"
echo "2. 检查生成的组织目录: $organized_path"
echo "3. 在思源笔记界面中使用AI功能时，系统现在能够针对性分析笔记本内容"
echo "4. 通过 API /api/notebook/searchContent 进行内容搜索"
echo ""
echo "故障排除:"
echo "- 如果API调用失败，检查思源笔记服务是否正常运行"
echo "- 如果权限不足，确保运行脚本的用户有读写数据目录的权限"
echo "- 如果数据不存在，先在思源笔记中创建一些笔记内容"