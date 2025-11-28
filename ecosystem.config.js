module.exports = {
    apps: [
        {
            name: 'siyuan-kernel',
            cwd: '/root/code/siyuan/kernel',
            script: './siyuan-kernel',
            args: '--mode production --port 6806 --workspace /root/code/siyuan/workspace',
            env: {
                SIYUAN_WORKSPACE: '/root/code/siyuan/workspace',
                SIYUAN_PORT: '6806',
                SIYUAN_WEB_MODE: 'true',
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
            error_file: '/tmp/siyuan-error.log',
            out_file: '/tmp/siyuan-out.log',
            log_date_format: 'YYYY-MM-DD HH:mm:ss Z',
            merge_logs: true,
        },
    ],
};
