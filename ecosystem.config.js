module.exports = {
    apps: [
        {
            name: 'siyuan-kernel',
            cwd: '/root/code/neu-siyuan-note',
            script: './siyuan-kernel',
            args: '--mode=2 --port=6806 --wd=/root/code/neu-siyuan-note --workspace=/root/code/MindOcean/user-data/notes',
            env: {
                SIYUAN_WORKSPACE: '/root/code/MindOcean/user-data/notes',
                SIYUAN_PORT: '6806',
                SIYUAN_WEB_MODE: 'true',
                SIYUAN_ACCESS_AUTH_CODE_BYPASS: 'true',
                // 用户数据根目录 - 每个用户的工作空间将在此目录下创建
                SIYUAN_USER_DATA_ROOT: '/root/code/MindOcean/user-data/notes',
                // AI 配置 - 可以根据需要设置
                // OPENAI_API_KEY: 'your-api-key',
                // SIYUAN_LLM_PROVIDER: 'openai',
                // SIYUAN_LLM_MODEL: 'gpt-4o-mini',
                // SIYUAN_EMBEDDING_PROVIDER: 'siliconflow',
                // SIYUAN_EMBEDDING_MODEL: 'BAAI/bge-large-zh-v1.5',
            },
            instances: 1,
            autorestart: true,
            watch: false,
            max_memory_restart: '2G',
            error_file: '/root/code/pm2-apps/logs/siyuan-error.log',
            out_file: '/root/code/pm2-apps/logs/siyuan-out.log',
            log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
            merge_logs: true,
        },
    ],
};
