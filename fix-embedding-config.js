#!/usr/bin/env node

/**
 * 修复向量化配置中的 API BaseURL
 * 从错误的 /v1/embeddings 改为正确的 /v1
 */

const fs = require('fs');
const path = require('path');

// 查找所有可能的配置文件位置
const possiblePaths = [
    path.join(process.env.HOME, '.config/siyuan/conf.json'),
    path.join(process.env.HOME, 'workspace/conf/conf.json'),
    '/root/workspace/conf/conf.json',
    '/home/jason/workspace/conf/conf.json',
];

// 遍历所有工作空间目录
const workspaceDir = '/root/code/siyuan/workspace';
if (fs.existsSync(workspaceDir)) {
    const users = fs.readdirSync(workspaceDir);
    users.forEach(user => {
        const confPath = path.join(workspaceDir, user, 'conf/conf.json');
        possiblePaths.push(confPath);
    });
}

console.log('🔍 搜索配置文件...\n');

let fixed = 0;

possiblePaths.forEach(confPath => {
    if (fs.existsSync(confPath)) {
        console.log(`📄 找到配置文件: ${confPath}`);

        try {
            const data = fs.readFileSync(confPath, 'utf8');
            const config = JSON.parse(data);

            // 检查是否有向量化配置
            if (config.ai && config.ai.embedding) {
                const oldBaseURL = config.ai.embedding.apiBaseUrl;
                console.log(`   当前 apiBaseUrl: ${oldBaseURL}`);

                // 修复错误的 URL
                if (oldBaseURL && oldBaseURL.includes('/embeddings')) {
                    config.ai.embedding.apiBaseUrl = oldBaseURL.replace('/embeddings', '');

                    // 保存修改
                    fs.writeFileSync(confPath, JSON.stringify(config, null, 2), 'utf8');
                    console.log(`   ✅ 已修复为: ${config.ai.embedding.apiBaseUrl}\n`);
                    fixed++;
                } else {
                    console.log(`   ℹ️  配置正确，无需修改\n`);
                }
            } else {
                console.log(`   ℹ️  无向量化配置\n`);
            }
        } catch (err) {
            console.error(`   ❌ 处理失败: ${err.message}\n`);
        }
    }
});

console.log(`\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
console.log(`✨ 完成！共修复 ${fixed} 个配置文件`);
console.log(`\n如果配置已修复，请重启思源笔记以应用更改。`);
