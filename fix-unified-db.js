// 修复统一注册服务数据库 - 创建 users 表
const Database = require('/home/jason/code/unified-settings-service/node_modules/better-sqlite3');
const path = require('path');

const dbPath = '/home/jason/code/unified-settings-service/database/settings.db';

console.log('🔧 开始修复统一注册服务数据库...');

try {
    const db = new Database(dbPath);

    // 检查现有表
    console.log('\n📋 检查现有表:');
    const tables = db.prepare("SELECT name FROM sqlite_master WHERE type='table'").all();
    console.log('当前表:', tables.map(t => t.name));

    // 创建 users 表
    console.log('\n🔨 创建 users 表...');
    db.exec(`
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            email TEXT UNIQUE NOT NULL,
            password TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            status TEXT DEFAULT 'active',
            workspace_path TEXT,
            is_active INTEGER DEFAULT 1
        )
    `);
    console.log('✅ users 表创建成功');

    // 检查表结构
    console.log('\n📊 users 表结构:');
    const columns = db.pragma('table_info(users)');
    columns.forEach(col => {
        console.log(`  - ${col.name}: ${col.type} ${col.notnull ? 'NOT NULL' : ''} ${col.pk ? 'PRIMARY KEY' : ''}`);
    });

    // 再次检查所有表
    console.log('\n📋 更新后的表列表:');
    const updatedTables = db.prepare("SELECT name FROM sqlite_master WHERE type='table'").all();
    console.log('所有表:', updatedTables.map(t => t.name));

    db.close();
    console.log('\n🎉 数据库修复完成！');

} catch (error) {
    console.error('❌ 修复失败:', error.message);
    console.error('详细错误:', error);
    process.exit(1);
}
