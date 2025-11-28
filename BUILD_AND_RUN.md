# 思源笔记构建和运行指南

## 📦 项目结构

```
/root/code/siyuan/
├── app/                      # 前端代码
│   ├── src/                 # 源代码
│   ├── stage/               # 构建输出
│   └── package.json
├── kernel/                   # 后端代码
│   ├── api/                 # API 实现
│   ├── main.go              # 主入口
│   ├── siyuan-kernel        # 编译后的二进制文件
│   └── go.mod
├── workspace/                # 工作空间（用户数据）
│   ├── conf/                # 配置文件
│   ├── data/                # 笔记数据
│   └── temp/                # 临时文件和数据库
├── ecosystem.config.js       # PM2 配置文件
├── start-production.sh       # 生产环境启动脚本
└── check-status.sh          # 服务状态检查脚本
```

## 🚀 快速开始

### 1. 构建项目

#### 构建前端
```bash
cd /root/code/siyuan/app
npm install              # 如果还没安装依赖
npm run build:app        # 生产构建
```

#### 构建后端
```bash
cd /root/code/siyuan/kernel
go mod tidy              # 整理依赖
CGO_ENABLED=1 go build -v -o siyuan-kernel -tags "fts5" -ldflags "-s -w" .
```

### 2. 启动服务

使用生产环境启动脚本（推荐）：
```bash
cd /root/code/siyuan
./start-production.sh
```

### 3. 检查服务状态

```bash
cd /root/code/siyuan
./check-status.sh
```

## 🔧 服务管理

### PM2 常用命令

```bash
# 查看所有服务
pm2 list

# 查看思源笔记日志
pm2 logs siyuan-kernel

# 查看实时日志
pm2 logs siyuan-kernel --lines 100

# 停止服务
pm2 stop siyuan-kernel

# 重启服务
pm2 restart siyuan-kernel

# 删除服务
pm2 delete siyuan-kernel

# 保存 PM2 配置（开机自启）
pm2 save
pm2 startup
```

## 🌐 访问应用

- **Web 界面**: http://localhost:6806
- **API 接口**: http://localhost:6806/api

### 首次访问

1. 打开浏览器访问 http://localhost:6806
2. 系统会提示创建账户或登录
3. 按照提示完成初始化设置

## 🤖 AI 功能配置

如需启用 AI 功能，编辑 `ecosystem.config.js` 文件，在 `env` 配置中添加：

```javascript
env: {
  // 基础配置
  SIYUAN_WORKSPACE: '/root/code/siyuan/workspace',
  SIYUAN_PORT: '6806',
  SIYUAN_WEB_MODE: 'true',
  
  // LLM 对话配置
  OPENAI_API_KEY: 'sk-your-api-key',              // OpenAI API 密钥
  SIYUAN_LLM_PROVIDER: 'openai',                  // LLM 提供商
  SIYUAN_LLM_MODEL: 'gpt-4o-mini',                // 模型名称
  SIYUAN_LLM_TEMPERATURE: '0.7',                  // 温度参数
  SIYUAN_LLM_MAX_TOKENS: '4000',                  // 最大令牌数
  
  // 向量化配置
  SIYUAN_EMBEDDING_API_KEY: 'sk-your-api-key',    // 向量化 API 密钥
  SIYUAN_EMBEDDING_PROVIDER: 'siliconflow',       // 向量化提供商
  SIYUAN_EMBEDDING_MODEL: 'BAAI/bge-large-zh-v1.5', // 向量化模型
},
```

配置完成后重启服务：
```bash
pm2 restart siyuan-kernel
```

### 推荐的 AI 模型

**LLM 对话模型：**
- OpenAI: `gpt-4o-mini` (快速经济), `gpt-4o` (强大)
- Anthropic: `claude-3-haiku` (快速), `claude-3-sonnet` (均衡)

**向量化模型：**
- SiliconFlow: `BAAI/bge-large-zh-v1.5` (中文), `BAAI/bge-m3` (多语言)
- OpenAI: `text-embedding-3-small` (经济), `text-embedding-3-large` (精确)

## 📝 开发模式

如果需要开发调试，可以使用开发模式：

```bash
# 前端开发模式（带热重载）
cd /root/code/siyuan/app
npm run dev

# 后端开发模式
cd /root/code/siyuan/kernel
go run main.go --mode development --port 6806
```

## 🔍 故障排查

### 服务无法启动

1. 检查构建产物是否存在：
   ```bash
   ls -lh /root/code/siyuan/kernel/siyuan-kernel
   ls -lh /root/code/siyuan/app/stage/build/
   ```

2. 检查端口是否被占用：
   ```bash
   netstat -tlnp | grep 6806
   ```

3. 查看详细日志：
   ```bash
   pm2 logs siyuan-kernel --lines 200
   ```

### 前端无法访问

1. 检查前端文件是否正确构建：
   ```bash
   ls -la /root/code/siyuan/app/stage/
   ```

2. 检查后端是否正确提供静态文件服务

### API 报错

1. 查看后端日志：
   ```bash
   pm2 logs siyuan-kernel
   ```

2. 检查工作空间目录权限：
   ```bash
   ls -ld /root/code/siyuan/workspace
   ```

## 📊 系统要求

- **操作系统**: Linux (Debian/Ubuntu)
- **Node.js**: 14.x 或更高
- **Go**: 1.20 或更高
- **内存**: 建议 2GB 以上
- **磁盘**: 建议 1GB 以上可用空间

## 🔄 更新和重新构建

```bash
# 1. 拉取最新代码（如果使用 Git）
cd /root/code/siyuan
git pull

# 2. 重新构建前端
cd app
npm install
npm run build:app

# 3. 重新构建后端
cd ../kernel
go mod tidy
CGO_ENABLED=1 go build -v -o siyuan-kernel -tags "fts5" -ldflags "-s -w" .

# 4. 重启服务
cd ..
pm2 restart siyuan-kernel
```

## 📚 相关文档

- [API 文档](./API_zh_CN.md)
- [AI 增强指南](./AI_ENHANCEMENT_GUIDE.md)
- [多用户 Web 应用指南](./WEB_MULTIUSER_GUIDE.md)

## 🆘 获取帮助

如果遇到问题：
1. 查看日志文件
2. 检查服务状态
3. 参考故障排查部分
4. 查阅相关文档

---

**版本**: 3.4.0  
**最后更新**: 2025-11-28
