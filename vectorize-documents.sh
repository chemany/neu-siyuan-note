#!/bin/bash

# 向量化指定的文档文件
# 这个脚本会直接调用灵枢笔记的 Go 代码来向量化文档

echo "========================================="
echo "文档向量化脚本"
echo "========================================="
echo ""

# 需要向量化的文档列表
DOCUMENTS=(
    "/root/code/MindOcean/user-data/notes/jason/assets/桃醛椰醛生产工艺专家咨询-0926 -20240816133118-zpxshbq.docx"
    "/root/code/MindOcean/user-data/notes/jason/assets/桃醛椰醛生产工艺专家咨询220919-20240816133118-nwtuyv0.docx"
    "/root/code/MindOcean/user-data/notes/jason/assets/咨询申请表 绿色化工-高新芊 2024-08-15-20240816133118-bv8bjq6.xlsx"
)

echo "检查文档是否存在..."
echo ""

for DOC in "${DOCUMENTS[@]}"; do
    if [ -f "$DOC" ]; then
        echo "✓ 找到: $(basename "$DOC")"
    else
        echo "✗ 未找到: $(basename "$DOC")"
    fi
done

echo ""
echo "========================================="
echo "开始向量化文档"
echo "========================================="
echo ""
echo "说明："
echo "由于灵枢笔记的 API 需要认证，我们需要通过前端界面来触发向量化。"
echo ""
echo "请按以下步骤操作："
echo ""
echo "1. 在浏览器中打开灵枢笔记"
echo "   https://www.cheman.top/notepads/stage/build/desktop"
echo ""
echo "2. 打开浏览器开发者工具（F12）"
echo ""
echo "3. 在控制台（Console）中执行以下代码："
echo ""
echo "=== 复制以下代码到浏览器控制台 ==="
echo ""
cat << 'EOF'
// 向量化文档的函数
async function vectorizeDocuments() {
    const documents = [
        "桃醛椰醛生产工艺专家咨询-0926 -20240816133118-zpxshbq.docx",
        "桃醛椰醛生产工艺专家咨询220919-20240816133118-nwtuyv0.docx",
        "咨询申请表 绿色化工-高新芊 2024-08-15-20240816133118-bv8bjq6.xlsx"
    ];
    
    console.log("开始向量化文档...");
    
    for (const doc of documents) {
        const assetPath = `/root/code/MindOcean/user-data/notes/jason/assets/${doc}`;
        console.log(`\n正在向量化: ${doc}`);
        
        try {
            const response = await fetch('/api/ai/vectorizeAsset', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ assetPath })
            });
            
            const result = await response.json();
            
            if (result.code === 0) {
                console.log(`✓ 向量化成功: ${doc}`);
            } else {
                console.error(`✗ 向量化失败: ${doc}`, result.msg);
            }
        } catch (error) {
            console.error(`✗ 请求失败: ${doc}`, error);
        }
        
        // 等待 2 秒再处理下一个文档
        await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
    console.log("\n向量化任务完成！");
}

// 执行向量化
vectorizeDocuments();
EOF
echo ""
echo "=== 代码结束 ==="
echo ""
echo "4. 等待向量化完成（大约需要 1-2 分钟）"
echo ""
echo "5. 完成后，可以重新尝试 AI 分析文档功能"
echo ""
echo "========================================="
