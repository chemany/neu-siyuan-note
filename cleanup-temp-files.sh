#!/bin/bash

# 灵枢笔记临时文件清理脚本
# 此脚本会删除开发过程中产生的临时文档和测试脚本

set -e

echo "========================================="
echo "灵枢笔记临时文件清理脚本"
echo "========================================="
echo ""

# 切换到脚本所在目录
cd "$(dirname "$0")"

# 统计要删除的文件数量
echo "正在统计要删除的文件..."
echo ""

MD_COUNT=$(ls -1 *_FIX*.md *_COMPLETE.md *_SUMMARY.md *_GUIDE.md *_MANUAL.md *_DESIGN.md *_IMPLEMENTATION.md *_INTEGRATION.md *_PROGRESS.md *_REFACTOR*.md REFACTOR*.md PHASE*.md MIGRATION*.md LOADTREE*.md 2>/dev/null | wc -l)
TEST_COUNT=$(ls -1 test-*.sh test-*.py test-*.js test-*.html test_*.sh test_*.py 2>/dev/null | wc -l)
TOOL_COUNT=$(ls -1 fix-*.sh fix-*.js fix-*.py diagnose-*.sh check-*.sh verify-*.sh batch-*.sh monitor-*.sh create-*.sh get-*.sh migrate-*.sh sync-*.js 2>/dev/null | wc -l)
CN_DOC_COUNT=$(ls -1 快速参考.md 灵枢笔记问题排查总结.md 修复完成报告.md 同步功能修复说明.md 密钥自动重新生成问题修复说明.md 最终修复方案.md 解决密钥不匹配问题.md 使用旧密钥同步.md 清空云端重新同步.md WEBDAV_问题排查总结.md WebDAV同步配置指南.md 2>/dev/null | wc -l)
CN_SCRIPT_COUNT=$(ls -1 查看*.sh 等待*.sh 监控*.sh 设置*.sh 实时*.sh 诊断*.sh 2>/dev/null | wc -l)
OTHER_COUNT=$(ls -1 API.md API_zh_CN.md README_ja_JP.md CHANGELOG.md CURRENT_STATUS.txt test-reranker.md check_db.py ocr_verify.go test-output.log 2>/dev/null | wc -l)

TOTAL=$((MD_COUNT + TEST_COUNT + TOOL_COUNT + CN_DOC_COUNT + CN_SCRIPT_COUNT + OTHER_COUNT))

echo "要删除的文件统计："
echo "  - 临时 MD 文档: $MD_COUNT 个"
echo "  - 测试脚本: $TEST_COUNT 个"
echo "  - 工具脚本: $TOOL_COUNT 个"
echo "  - 中文文档: $CN_DOC_COUNT 个"
echo "  - 中文脚本: $CN_SCRIPT_COUNT 个"
echo "  - 其他临时文件: $OTHER_COUNT 个"
echo "  --------------------------------"
echo "  总计: $TOTAL 个文件"
echo ""

# 保留的核心文档
echo "保留的核心文档："
echo "  ✓ README.md"
echo "  ✓ LICENSE"
echo "  ✓ BUILD_AND_DEPLOY.md"
echo "  ✓ BLOCKTREE_ARCHITECTURE.md"
echo "  ✓ SUBDOCUMENT_CREATION_FLOW.md"
echo "  ✓ Dockerfile"
echo "  ✓ go.mod"
echo "  ✓ ecosystem.config.js"
echo ""

# 保留但不提交的文件
echo "保留但不提交的文件："
echo "  ✓ rebuild-*.sh (构建脚本)"
echo "  ✓ reload-*.sh (重载脚本)"
echo ""

# 确认删除
read -p "是否继续删除这些文件？(y/N) " -n 1 -r
echo ""

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "取消删除操作"
    exit 0
fi

echo ""
echo "开始删除文件..."
echo ""

# 1. 删除临时 MD 文档
echo "[1/6] 删除临时 MD 文档..."
rm -f *_FIX*.md *_COMPLETE.md *_SUMMARY.md *_GUIDE.md *_MANUAL.md \
      *_DESIGN.md *_IMPLEMENTATION.md *_INTEGRATION.md *_PROGRESS.md \
      *_REFACTOR*.md REFACTOR*.md PHASE*.md MIGRATION*.md LOADTREE*.md \
      2>/dev/null || true
echo "  ✓ 已删除 $MD_COUNT 个临时 MD 文档"

# 2. 删除测试脚本
echo "[2/6] 删除测试脚本..."
rm -f test-*.sh test-*.py test-*.js test-*.html test_*.sh test_*.py \
      2>/dev/null || true
echo "  ✓ 已删除 $TEST_COUNT 个测试脚本"

# 3. 删除工具脚本
echo "[3/6] 删除工具脚本..."
rm -f fix-*.sh fix-*.js fix-*.py diagnose-*.sh check-*.sh verify-*.sh \
      batch-*.sh monitor-*.sh create-*.sh get-*.sh migrate-*.sh sync-*.js \
      2>/dev/null || true
echo "  ✓ 已删除 $TOOL_COUNT 个工具脚本"

# 4. 删除中文文档
echo "[4/6] 删除中文文档..."
rm -f 快速参考.md 灵枢笔记问题排查总结.md 修复完成报告.md \
      同步功能修复说明.md 密钥自动重新生成问题修复说明.md \
      最终修复方案.md 解决密钥不匹配问题.md 使用旧密钥同步.md \
      清空云端重新同步.md WEBDAV_问题排查总结.md WebDAV同步配置指南.md \
      2>/dev/null || true
echo "  ✓ 已删除 $CN_DOC_COUNT 个中文文档"

# 5. 删除中文脚本
echo "[5/6] 删除中文脚本..."
rm -f 查看*.sh 等待*.sh 监控*.sh 设置*.sh 实时*.sh 诊断*.sh \
      2>/dev/null || true
echo "  ✓ 已删除 $CN_SCRIPT_COUNT 个中文脚本"

# 6. 删除其他临时文件
echo "[6/6] 删除其他临时文件..."
rm -f API.md API_zh_CN.md README_ja_JP.md CHANGELOG.md CURRENT_STATUS.txt \
      test-reranker.md check_db.py ocr_verify.go test-output.log \
      2>/dev/null || true
echo "  ✓ 已删除 $OTHER_COUNT 个其他临时文件"

echo ""
echo "========================================="
echo "✓ 清理完成！"
echo "========================================="
echo ""
echo "已删除 $TOTAL 个临时文件"
echo ""
echo "保留的核心文档："
ls -1 README.md LICENSE BUILD_AND_DEPLOY.md BLOCKTREE_ARCHITECTURE.md \
     SUBDOCUMENT_CREATION_FLOW.md Dockerfile go.mod ecosystem.config.js \
     2>/dev/null || true
echo ""
echo "提示："
echo "  - 所有删除的文件都在 Git 历史中，如需恢复可以使用 git"
echo "  - 已更新 .gitignore，未来不会再提交这些临时文件"
echo "  - 构建脚本（rebuild-*.sh）已保留，但不会提交到 Git"
echo ""
