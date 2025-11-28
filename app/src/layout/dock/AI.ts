/// #if !MOBILE
import { Tab } from "../Tab";
import { setPanelFocus } from "../util";
/// #endif
import { Model } from "../Model";
import { App } from "../../index";
import { updateHotkeyAfterTip } from "../../protyle/util/compatibility";
import { getDockByType } from "../tabUtil";
import { getAllModels } from "../getAll";
import { insertHTML } from "../../protyle/util/insertHTML";
import { focusBlock } from "../../protyle/util/selection";

interface IAIMessage {
    role: "user" | "assistant" | "system";
    content: string;
    timestamp: number;
}

export class AI extends Model {
    private element: Element;
    private messages: IAIMessage[] = [];
    private currentEditor: any = null;

    constructor(app: App, tab: Tab | Element) {
        super({ app, id: tab.id });
        if (tab instanceof Element) {
            this.element = tab;
        } else {
            this.element = tab.panelElement;
        }

        /// #if MOBILE
        this.element.innerHTML = `<div class="toolbar toolbar--border toolbar--dark">
    <div class="fn__space"></div>
    <div class="toolbar__text">
        AI 文档分析
    </div>
    <span class="fn__flex-1"></span>
</div>
<div class="fn__flex-1 ai-chat-container" style="background-color: var(--b3-theme-background); padding: 10px; display: flex; flex-direction: column;">
    <div class="ai-messages" style="flex: 1; overflow-y: auto; margin-bottom: 10px; border: 1px solid var(--b3-theme-surface-lighter); border-radius: 4px; padding: 10px;" data-type="messages">
        <div class="ai-welcome" style="color: var(--b3-theme-on-surface-light); text-align: center; padding: 20px;">
            <p style="margin-bottom: 10px;">🤖 AI 文档分析助手</p>
            <p style="font-size: 12px;">选择一个提示词快速开始分析当前文档</p>
        </div>
    </div>
    <div class="ai-prompts" style="margin-bottom: 8px; display: flex; flex-wrap: wrap; gap: 4px;">
        <button class="b3-button b3-button--outline" data-prompt="总结" style="font-size: 12px; padding: 2px 8px;">📝 总结文档</button>
        <button class="b3-button b3-button--outline" data-prompt="要点" style="font-size: 12px; padding: 2px 8px;">🎯 提取要点</button>
        <button class="b3-button b3-button--outline" data-prompt="续写" style="font-size: 12px; padding: 2px 8px;">✍️ 续写</button>
        <button class="b3-button b3-button--outline" data-prompt="优化" style="font-size: 12px; padding: 2px 8px;">✨ 优化</button>
    </div>
    <div class="ai-input" style="display: flex; gap: 4px;">
        <input type="text" class="b3-text-field fn__flex-1" placeholder="输入问题或选择提示词..." data-type="input">
        <button class="b3-button b3-button--outline" data-type="send">发送</button>
    </div>
</div>`;
        /// #else
        this.element.classList.add("fn__flex-column", "file-tree", "sy__ai");
        this.element.innerHTML = `<div class="block__icons">
    <div class="block__logo">
        <svg class="block__logoicon"><use xlink:href="#iconSparkles"></use></svg>AI 文档分析&nbsp;
    </div>
    <span class="fn__flex-1"></span>
    <span data-type="min" class="block__icon b3-tooltips b3-tooltips__w" aria-label="${window.siyuan.languages.min}${updateHotkeyAfterTip(window.siyuan.config.keymap.general.closeTab.custom)}"><svg><use xlink:href="#iconMin"></use></svg></span>
</div>
<div class="fn__flex-1 ai-chat-container" style="background-color: var(--b3-theme-background); padding: 8px; display: flex; flex-direction: column;">
    <div class="ai-messages" style="flex: 1; overflow-y: auto; margin-bottom: 8px; border: 1px solid var(--b3-border-color); border-radius: 4px; padding: 8px; background: var(--b3-theme-surface);" data-type="messages">
        <div class="ai-welcome" style="color: var(--b3-theme-on-surface-light); text-align: center; padding: 20px 10px;">
            <div style="font-size: 24px; margin-bottom: 8px;">🤖</div>
            <div style="font-weight: bold; margin-bottom: 8px;">AI 文档分析助手</div>
            <div style="font-size: 12px; line-height: 1.6;">
                选择一个提示词快速开始分析当前文档<br>
                分析完成后可以保存到笔记末尾
            </div>
        </div>
    </div>
    <div class="ai-prompts" style="margin-bottom: 8px; display: grid; grid-template-columns: 1fr 1fr; gap: 6px;">
        <button class="b3-button b3-button--outline" data-prompt="总结" style="font-size: 12px;">📝 总结文档</button>
        <button class="b3-button b3-button--outline" data-prompt="要点" style="font-size: 12px;">🎯 提取要点</button>
        <button class="b3-button b3-button--outline" data-prompt="续写" style="font-size: 12px;">✍️ 续写内容</button>
        <button class="b3-button b3-button--outline" data-prompt="优化" style="font-size: 12px;">✨ 优化表达</button>
        <button class="b3-button b3-button--outline" data-prompt="翻译" style="font-size: 12px;">🌐 翻译</button>
        <button class="b3-button b3-button--outline" data-prompt="问答" style="font-size: 12px;">💬 问答</button>
    </div>
    <div class="ai-input-container" style="display: flex; flex-direction: column; gap: 6px;">
        <div class="ai-input" style="display: flex; gap: 6px;">
            <input type="text" class="b3-text-field fn__flex-1" placeholder="输入问题或选择提示词..." data-type="input" style="font-size: 13px;">
            <button class="b3-button b3-button--outline" data-type="send" style="min-width: 60px;">发送</button>
        </div>
        <div class="ai-actions" style="display: none; gap: 6px;">
            <button class="b3-button b3-button--outline fn__flex-1" data-type="save" style="font-size: 12px;">💾 保存到笔记</button>
            <button class="b3-button b3-button--text" data-type="clear" style="font-size: 12px;">🗑️ 清空</button>
        </div>
    </div>
</div>`;
        /// #endif

        this.bindEvents();
    }

    private bindEvents() {
        this.element.addEventListener("click", (event: MouseEvent) => {
            /// #if !MOBILE
            setPanelFocus(this.element);
            /// #endif
            let target = event.target as HTMLElement;
            while (target && !target.isEqualNode(this.element)) {
                const type = target.getAttribute("data-type");
                const prompt = target.getAttribute("data-prompt");

                if (type === "min") {
                    getDockByType("ai").toggleModel("ai", false, true);
                    event.preventDefault();
                    break;
                } else if (type === "send") {
                    this.handleSend();
                    event.preventDefault();
                    break;
                } else if (type === "save") {
                    this.saveToNote();
                    event.preventDefault();
                    break;
                } else if (type === "clear") {
                    this.clearMessages();
                    event.preventDefault();
                    break;
                } else if (prompt) {
                    this.handlePromptClick(prompt);
                    event.preventDefault();
                    break;
                }
                target = target.parentElement;
            }
        });

        // 回车发送
        const inputElement = this.element.querySelector('[data-type="input"]') as HTMLInputElement;
        if (inputElement) {
            inputElement.addEventListener("keydown", (event: KeyboardEvent) => {
                if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    this.handleSend();
                }
            });
        }
    }

    private handlePromptClick(promptType: string) {
        const inputElement = this.element.querySelector('[data-type="input"]') as HTMLInputElement;
        const promptTexts: { [key: string]: string } = {
            "总结": "请总结这篇文档的主要内容",
            "要点": "请提取这篇文档的关键要点",
            "续写": "请根据当前内容继续写作",
            "优化": "请优化这篇文档的表达和结构",
            "翻译": "请将这篇文档翻译成英文",
            "问答": "请回答关于这篇文档的问题："
        };

        if (inputElement && promptTexts[promptType]) {
            inputElement.value = promptTexts[promptType];
            inputElement.focus();

            // 如果不是问答类型，直接发送
            if (promptType !== "问答") {
                setTimeout(() => this.handleSend(), 100);
            }
        }
    }

    private handleSend() {
        const inputElement = this.element.querySelector('[data-type="input"]') as HTMLInputElement;
        if (!inputElement || !inputElement.value.trim()) {
            return;
        }

        const userMessage = inputElement.value.trim();
        inputElement.value = "";

        // 添加用户消息
        this.addMessage("user", userMessage);

        // 获取当前文档内容
        const docContent = this.getCurrentDocContent();

        // 显示加载状态
        const loadingMsg = "🤔 正在思考中...";
        this.addMessage("assistant", loadingMsg);

        // 调用真实的AI API
        this.callAI(userMessage, docContent).then(aiResponse => {
            // 移除加载消息
            this.messages.pop();
            // 添加真实的AI回复
            this.addMessage("assistant", aiResponse);

            // 显示保存按钮
            const actionsElement = this.element.querySelector('.ai-actions') as HTMLElement;
            if (actionsElement) {
                actionsElement.style.display = "flex";
            }
        }).catch(error => {
            // 移除加载消息
            this.messages.pop();
            // 显示错误信息
            const errorMsg = `❌ AI调用失败: ${error.message || '未知错误'}\n\n请检查AI配置是否正确。`;
            this.addMessage("assistant", errorMsg);
            console.error('AI调用失败:', error);
        });
    }

    private getCurrentDocContent(): string {
        // 获取当前激活的编辑器
        const models = getAllModels();
        const activeEditor = models.editor.find(item =>
            item.parent?.headElement?.classList.contains("item--focus")
        );

        if (activeEditor && activeEditor.editor?.protyle) {
            this.currentEditor = activeEditor.editor;
            const wysiwygElement = activeEditor.editor.protyle.wysiwyg.element;
            return wysiwygElement.textContent || "";
        }

        return "";
    }

    private async callAI(question: string, docContent: string): Promise<string> {
        const messages = [];

        if (docContent && docContent.trim()) {
            messages.push({
                role: "system",
                content: `当前文档内容：\n\n${docContent.substring(0, 3000)}${docContent.length > 3000 ? '...' : ''}`
            });
        }

        messages.push({
            role: "user",
            content: question
        });

        try {
            let response;
            const isBuiltin = window.siyuan.config.ai.openAI.apiProvider === "builtin";

            if (isBuiltin) {
                const unifiedAuthUrl = window.siyuan.config.system.unifiedAuthServiceUrl || 'http://localhost:3002';
                const token = localStorage.getItem('siyuan_jwt_token');
                const headers: any = { 'Content-Type': 'application/json' };
                if (token) {
                    headers['Authorization'] = `Bearer ${token}`;
                }

                response = await fetch(`${unifiedAuthUrl}/api/ai/chat`, {
                    method: 'POST',
                    headers: headers,
                    body: JSON.stringify({
                        messages: messages,
                        stream: false
                    })
                });
            } else {
                // Fallback for standard Siyuan backend
                let combinedMsg = question;
                if (docContent && docContent.trim()) {
                    combinedMsg = `当前文档内容：\n\n${docContent.substring(0, 3000)}\n\n用户问题：${question}`;
                }

                response = await fetch('/api/ai/chatGPT', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        msg: combinedMsg
                    })
                });
            }

            if (!response.ok) {
                const errorData = await response.json().catch(() => ({}));
                throw new Error(errorData.msg || `HTTP ${response.status}`);
            }

            const result = await response.json();
            if (result.code !== 0) {
                throw new Error(result.msg || "AI服务返回错误");
            }

            // Handle different response structures
            return result.data?.content || result.data?.message || result.data || "抱歉，AI没有返回有效内容";

        } catch (error) {
            console.error("AI API调用失败:", error);
            throw error;
        }
    }

    private generateMockResponse(question: string, docContent: string): string {
        // 保留作为fallback，但不再使用
        // 这是一个模拟响应，实际应用中应该调用真实的AI API
        if (question.includes("总结")) {
            return `📋 **文档总结**\n\n根据当前文档内容，主要讨论了以下几个方面：\n\n1. 核心观点和主题\n2. 关键论据和支撑材料\n3. 结论和启示\n\n_（这是一个示例响应，实际应用中需要接入AI服务）_`;
        } else if (question.includes("要点")) {
            return `🎯 **关键要点**\n\n• 要点一：核心概念说明\n• 要点二：重要论据\n• 要点三：实践应用\n• 要点四：注意事项\n\n_（这是一个示例响应，实际应用中需要接入AI服务）_`;
        } else if (question.includes("续写")) {
            return `✍️ **续写建议**\n\n基于当前内容，可以从以下角度继续展开：\n\n1. 深入分析现有观点\n2. 补充相关案例\n3. 提出可能的解决方案\n4. 总结和展望\n\n_（这是一个示例响应，实际应用中需要接入AI服务）_`;
        } else if (question.includes("优化")) {
            return `✨ **优化建议**\n\n**结构优化：**\n- 建议调整段落顺序，使逻辑更清晰\n- 可以添加小标题，增强可读性\n\n**表达优化：**\n- 部分句子可以更简洁\n- 专业术语需要适当解释\n\n_（这是一个示例响应，实际应用中需要接入AI服务）_`;
        } else {
            return `💡 **AI 回复**\n\n关于您的问题"${question}"：\n\n根据文档内容分析，我的理解是...\n\n_（这是一个示例响应，实际应用中需要接入真实的AI服务进行分析）_`;
        }
    }

    private addMessage(role: "user" | "assistant" | "system", content: string) {
        const message: IAIMessage = {
            role,
            content,
            timestamp: Date.now()
        };

        this.messages.push(message);
        this.renderMessages();
    }

    private renderMessages() {
        const messagesContainer = this.element.querySelector('[data-type="messages"]');
        if (!messagesContainer) return;

        // 移除欢迎消息
        const welcomeElement = messagesContainer.querySelector('.ai-welcome');
        if (welcomeElement) {
            welcomeElement.remove();
        }

        // 清空并重新渲染所有消息
        messagesContainer.innerHTML = this.messages.map(msg => {
            const isUser = msg.role === "user";
            const bgColor = isUser ? "var(--b3-theme-primary-lighter)" : "var(--b3-theme-surface)";
            const align = isUser ? "flex-end" : "flex-start";
            const icon = isUser ? "👤" : "🤖";

            return `
                <div style="display: flex; justify-content: ${align}; margin-bottom: 12px;">
                    <div style="max-width: 85%; background: ${bgColor}; padding: 8px 12px; border-radius: 8px; word-wrap: break-word;">
                        <div style="font-size: 11px; color: var(--b3-theme-on-surface-light); margin-bottom: 4px;">
                            ${icon} ${isUser ? "我" : "AI助手"}
                        </div>
                        <div style="line-height: 1.6; white-space: pre-wrap; font-size: 13px;">${this.escapeHtml(msg.content)}</div>
                    </div>
                </div>
            `;
        }).join("");

        // 滚动到底部
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    private escapeHtml(text: string): string {
        // 支持简单的markdown格式
        return text
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>')
            .replace(/`(.*?)`/g, '<code style="background: var(--b3-theme-surface-lighter); padding: 2px 4px; border-radius: 2px;">$1</code>')
            .replace(/\n/g, '<br>');
    }

    private saveToNote() {
        if (!this.currentEditor || !this.currentEditor.protyle) {
            window.siyuan.showMessage?.("请先打开一个文档", 3000, "error");
            return;
        }

        // 获取最后一条AI回复
        const lastAIMessage = [...this.messages].reverse().find(msg => msg.role === "assistant");
        if (!lastAIMessage) {
            window.siyuan.showMessage?.("没有可保存的AI回复", 3000, "error");
            return;
        }

        try {
            const protyle = this.currentEditor.protyle;
            const lastBlock = protyle.wysiwyg.element.lastElementChild;

            if (lastBlock) {
                // 准备要插入的内容
                const insertContent = `\n\n---\n\n## 🤖 AI 分析结果\n\n${lastAIMessage.content}\n\n*生成时间：${new Date(lastAIMessage.timestamp).toLocaleString()}*\n`;

                // 使用 insertHTML 插入内容
                const htmlContent = protyle.lute.Md2BlockDOM(insertContent);
                insertHTML(htmlContent, protyle, true);

                // 聚焦到最后一个块
                setTimeout(() => {
                    const newLastBlock = protyle.wysiwyg.element.lastElementChild;
                    if (newLastBlock) {
                        focusBlock(newLastBlock, undefined, false);
                    }
                }, 100);

                window.siyuan.showMessage?.("✅ 已保存到笔记末尾", 2000, "info");
            }
        } catch (e) {
            console.error("保存到笔记失败:", e);
            window.siyuan.showMessage?.("保存失败，请重试", 3000, "error");
        }
    }

    private clearMessages() {
        this.messages = [];
        const messagesContainer = this.element.querySelector('[data-type="messages"]');
        if (messagesContainer) {
            messagesContainer.innerHTML = `
                <div class="ai-welcome" style="color: var(--b3-theme-on-surface-light); text-align: center; padding: 20px 10px;">
                    <div style="font-size: 24px; margin-bottom: 8px;">🤖</div>
                    <div style="font-weight: bold; margin-bottom: 8px;">AI 文档分析助手</div>
                    <div style="font-size: 12px; line-height: 1.6;">
                        选择一个提示词快速开始分析当前文档<br>
                        分析完成后可以保存到笔记末尾
                    </div>
                </div>
            `;
        }

        // 隐藏保存按钮
        const actionsElement = this.element.querySelector('.ai-actions') as HTMLElement;
        if (actionsElement) {
            actionsElement.style.display = "none";
        }
    }
}
