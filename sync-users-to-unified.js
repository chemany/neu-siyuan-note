// 同步思源笔记用户到统一注册服务
const fs = require('fs');
const Database = require('/home/jason/code/unified-settings-service/node_modules/better-sqlite3');
const bcrypt = require('/home/jason/code/unified-settings-service/node_modules/bcryptjs');

const siyuanUsersPath = '/root/code/siyuan/kernel/data/users/users.json';
const unifiedDBPath = '/home/jason/code/unified-settings-service/database/settings.db';
const unifiedCSVPath = '/home/jason/code/unified-settings-service/user-data-v2/users.csv';

console.log('🔄 开始同步用户数据...\n');

async function syncUsers() {
    try {
        // 1. 读取思源笔记用户数据
        console.log('📖 读取思源笔记用户数据...');
        const siyuanUsers = JSON.parse(fs.readFileSync(siyuanUsersPath, 'utf8'));
        console.log(`找到 ${siyuanUsers.length} 个用户:`, siyuanUsers.map(u => u.email).join(', '));

        // 2. 连接数据库
        console.log('\n📊 连接统一注册服务数据库...');
        const db = new Database(unifiedDBPath);

        // 3. 默认密码: zhangli1115
        const defaultPassword = 'zhangli1115';
        const hashedPassword = await bcrypt.hash(defaultPassword, 10);
        console.log(`使用默认密码: ${defaultPassword}`);

        // 4. 同步每个用户
        console.log('\n🔨 同步用户到数据库和CSV...');
        const csvLines = ['user_id,username,email,password,created_at,updated_at,status'];

        for (const user of siyuanUsers) {
            // 插入数据库
            try {
                const stmt = db.prepare(`
                    INSERT OR REPLACE INTO users (id, username, email, password, created_at, updated_at, status, workspace_path, is_active)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                `);

                stmt.run(
                    user.id,
                    user.username,
                    user.email,
                    hashedPassword,
                    user.created_at,
                    user.updated_at,
                    'active',
                    user.workspace || null,
                    user.is_active ? 1 : 0
                );

                console.log(`  ✅ ${user.email} -> 数据库`);
            } catch (dbError) {
                console.error(`  ❌ ${user.email} 数据库插入失败:`, dbError.message);
            }

            // CSV行
            csvLines.push(`${user.id},${user.username},${user.email},${hashedPassword},${user.created_at},${user.updated_at},active`);
            console.log(`  ✅ ${user.email} -> CSV`);
        }

        // 5. 写入CSV文件
        fs.writeFileSync(unifiedCSVPath, csvLines.join('\n') + '\n', 'utf8');
        console.log(`\n✅ CSV文件已更新: ${unifiedCSVPath}`);

        // 6. 验证数据库
        console.log('\n📋 验证数据库用户:');
        const dbUsers = db.prepare('SELECT id, username, email FROM users').all();
        dbUsers.forEach(u => {
            console.log(`  - ${u.email} (用户名: ${u.username}, ID: ${u.id})`);
        });

        db.close();

        console.log('\n🎉 用户数据同步完成！');
        console.log(`\n💡 测试登录: link918@qq.com / ${defaultPassword}`);

    } catch (error) {
        console.error('❌ 同步失败:', error);
        process.exit(1);
    }
}

syncUsers();
