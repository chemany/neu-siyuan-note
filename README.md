<p align="center">
<img alt="灵枢笔记" src="https://b3log.org/images/brand/siyuan-128.png">
<br>
<em>灵枢笔记 - 基于思源笔记的 Web 端优化版本</em>
<br><br>
<a title="License" target="_blank" href="https://www.gnu.org/licenses/agpl-3.0.txt"><img src="http://img.shields.io/badge/license-AGPLv3-orange.svg?style=flat-square"></a>
</p>

<p align="center">
<a href="https://www.cheman.top">官网</a> | <a href="https://www.cheman.top/notepads/">在线体验</a> | <a href="mailto:125607565@qq.com">联系我们</a>
</p>

---

## 📖 目录

* [💡 简介](#-简介)
* [✨ 特性优化](#-特性优化)
* [🚀 部署方式](#-部署方式)
* [📝 问题反馈](#-问题反馈)
* [🙏 致谢](#-致谢)

---

## 💡 简介

灵枢笔记是基于 [思源笔记 (SiYuan)](https://github.com/siyuan-note/siyuan) 开源项目进行二次开发的 Web 端优化版本。专注于提供更流畅的 Web 端使用体验，针对 Web 部署场景进行了多项优化和改进。

本项目保留了思源笔记的核心功能，同时针对 Web 端的特殊需求进行了定制化开发。

## ✨ 特性优化

### 🤖 AI & OCR 增强

- **私有大模型集成**：支持对接私有化部署的大语言模型（如通过 DeepSeek、LocalTVM 等），提供智能问答与内容生成，充分保障数据隐私。
- **RAG (检索增强生成)**：内置 RAG 引擎，在 AI 聊天时自动检索相关笔记内容作为背景知识，让 AI 更加了解你的个人知识库。
- **OCR 识别优化**：深度集成私有 OCR 服务，支持扫描版 PDF 和图片的文字识别，识别结果自动进入全文索引，实现“图片即搜索”。
- **语义搜索**：基于向量化技术，支持通过语义理解进行笔记搜索，即使关键词不完全匹配也能找到相关内容。

### 🔐 Web 端认证优化

- **JWT Token 认证**：支持通过 URL 参数、Cookie 或 localStorage 传递认证 Token
- **统一设置服务集成**：与统一设置服务 (unified-settings-service) 无缝集成，实现跨应用单点登录
- **访问码认证**：保留原有访问码认证方式，兼容多种认证场景

### 📁 文档树实时更新

- **新建笔记即时显示**：创建新笔记后，文档树立即更新显示，无需手动刷新
- **新建笔记本即时显示**：创建新笔记本后，文档树立即更新，不依赖 WebSocket 推送
- **删除笔记本即时更新**：删除笔记本后，文档树立即移除对应项目
- **笔记标题实时同步**：编辑笔记标题时，文档树中的标题实时更新

### 🎨 界面优化

- **顶部菜单栏**：新增顶部菜单栏，提供更便捷的操作入口
- **设置按钮**：在顶部菜单栏添加设置快捷入口
- **自定义反馈链接**：问题反馈链接指向自定义地址

### 🔧 技术优化

- **WebSocket 连接优化**：优化 WebSocket 连接的认证流程，支持 Token 参数传递
- **API 回调增强**：关键操作（如创建/删除笔记本）增加回调处理，确保 UI 及时更新
- **内存占用优化**：针对服务器部署场景优化内存使用

## 🚀 部署方式

### 环境要求

- Node.js 18+
- Go 1.21+
- pnpm

### 构建步骤

```bash
# 克隆项目
git clone https://github.com/your-repo/lingshu-note.git
cd lingshu-note

# 安装前端依赖
cd app
pnpm install

# 构建前端
pnpm run build:desktop

# 构建后端
cd ../kernel
go build -o siyuan-kernel

# 运行
./siyuan-kernel --mode production --port 6806 --workspace /path/to/workspace
```

### PM2 部署

```bash
pm2 start siyuan-kernel --name "lingshu-note" -- --mode production --port 6806 --workspace /path/to/workspace
```

### 配合 Nginx 反向代理

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:6806;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /ws {
        proxy_pass http://127.0.0.1:6806;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 📝 问题反馈

如果您在使用过程中遇到问题或有功能建议，欢迎通过以下方式联系我们：

- 🌐 官网：[www.cheman.top](https://www.cheman.top)
- 📧 邮箱：[125607565@qq.com](mailto:125607565@qq.com?subject=灵枢笔记问题反馈)

## 🙏 致谢

灵枢笔记的诞生离不开以下开源项目和贡献者：

### 思源笔记 (SiYuan)

本项目基于 [思源笔记 (SiYuan)](https://github.com/siyuan-note/siyuan) 开源项目进行二次开发。思源笔记是一款隐私优先的个人知识管理系统，支持细粒度块级引用和 Markdown 所见即所得。

**特别感谢思源笔记团队的开源贡献！**

- 🔗 思源笔记官网：[https://b3log.org/siyuan](https://b3log.org/siyuan)
- 📦 思源笔记 GitHub：[https://github.com/siyuan-note/siyuan](https://github.com/siyuan-note/siyuan)
- 💬 思源笔记社区：[https://ld246.com](https://ld246.com)

### 相关开源项目

| 项目 | 描述 |
|------|------|
| [lute](https://github.com/88250/lute) | 编辑器引擎 |
| [dejavu](https://github.com/siyuan-note/dejavu) | 数据仓库 |
| [petal](https://github.com/siyuan-note/petal) | 插件 API |

### 开源协议

本项目遵循 [AGPLv3](https://www.gnu.org/licenses/agpl-3.0.txt) 开源协议，与思源笔记保持一致。

---

<p align="center">
<em>感谢使用灵枢笔记 ❤️</em>
</p>
