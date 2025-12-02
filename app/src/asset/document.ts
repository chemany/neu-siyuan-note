import {Model} from "../layout/Model";
import {Tab} from "../layout/Tab";
import {setPanelFocus} from "../layout/util";
import {App} from "../index";
import {clearOBG} from "../layout/dock/util";
import {pathPosix} from "../util/pathName";
import * as mammoth from "mammoth";

export class DocumentViewer extends Model {
    public path: string;
    public element: HTMLElement;

    constructor(options: { app: App, tab: Tab, path: string }) {
        super({app: options.app, id: options.tab.id});
        if (window.siyuan.config.fileTree.openFilesUseCurrentTab) {
            options.tab.headElement.classList.add("item--unupdate");
        }
        this.element = options.tab.panelElement;
        this.path = options.path;
        this.element.addEventListener("click", () => {
            clearOBG();
            setPanelFocus(this.element.parentElement.parentElement);
        });
        this.render();
    }

    private getPublicFileUrl(): string {
        const currentHost = window.location.host;
        let publicHost = currentHost;
        // 如果是本地访问，使用公网 IP
        if (currentHost.includes("localhost") || currentHost.includes("127.0.0.1")) {
            publicHost = "156.233.228.53:6806";
        }
        const protocol = window.location.protocol;
        if (this.path.startsWith("http://") || this.path.startsWith("https://")) {
            return this.path;
        } else if (this.path.startsWith("file://")) {
            return null;
        }
        // 使用公开的 assets 端点，将 assets/ 替换为 public-assets/
        let publicPath = this.path;
        if (publicPath.startsWith("assets/")) {
            publicPath = "public-assets/" + publicPath.substring(7);
        } else if (publicPath.startsWith("/assets/")) {
            publicPath = "/public-assets/" + publicPath.substring(8);
        }
        // 对文件名进行 URL 编码（保留路径分隔符）
        const parts = publicPath.split("/");
        const encodedParts = parts.map((part, index) => {
            // 最后一部分是文件名，需要编码
            if (index === parts.length - 1) {
                return encodeURIComponent(part);
            }
            return part;
        });
        const encodedPath = encodedParts.join("/");
        const pathStr = encodedPath.startsWith("/") ? encodedPath : "/" + encodedPath;
        return protocol + "//" + publicHost + pathStr;
    }

    private getLocalFileUrl(): string {
        // 如果已经是完整 URL，直接返回
        if (this.path.startsWith("http://") || this.path.startsWith("https://")) {
            return this.path;
        }
        // 如果是 file:// 协议
        if (this.path.startsWith("file://")) {
            return this.path;
        }
        // 相对路径，使用当前 origin
        const origin = window.location.origin;
        // 确保路径以 / 开头
        const path = this.path.startsWith("/") ? this.path : "/" + this.path;
        console.log("[DocumentViewer] path:", this.path, "url:", origin + path);
        return origin + path;
    }

    private render() {
        const ext = pathPosix().extname(this.path).toLowerCase().split("?")[0];
        const localUrl = this.getLocalFileUrl();
        const textExts = [".md", ".markdown", ".txt", ".json", ".xml", ".css", ".js", ".ts", ".py", ".go", ".java", ".c", ".cpp", ".h", ".sh", ".yaml", ".yml", ".toml", ".ini", ".conf", ".log"];
        const officeExts = [".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp"];
        
        if (textExts.includes(ext)) {
            this.renderTextFile(localUrl, ext);
        } else if (officeExts.includes(ext)) {
            this.renderOfficeFile(localUrl);
        } else {
            this.renderGenericFile(localUrl);
        }
    }

    private async renderTextFile(fileUrl: string, ext: string) {
        const fileName = this.getFileName();
        this.element.innerHTML = "";
        
        const container = document.createElement("div");
        container.style.cssText = "width:100%;height:100%;display:flex;flex-direction:column;background:var(--b3-theme-background)";
        
        const toolbar = document.createElement("div");
        toolbar.style.cssText = "padding:8px 16px;background:var(--b3-theme-surface);border-bottom:1px solid var(--b3-border-color);display:flex;align-items:center;gap:8px";
        
        const title = document.createElement("span");
        title.style.cssText = "color:var(--b3-theme-on-surface);font-size:13px;font-weight:500";
        title.textContent = fileName;
        toolbar.appendChild(title);
        
        const spacer = document.createElement("div");
        spacer.style.flex = "1";
        toolbar.appendChild(spacer);
        
        const downloadBtn = document.createElement("a");
        downloadBtn.href = fileUrl;
        downloadBtn.download = "";
        downloadBtn.className = "b3-button b3-button--small b3-button--outline";
        downloadBtn.style.textDecoration = "none";
        downloadBtn.textContent = "下载";
        toolbar.appendChild(downloadBtn);
        
        const content = document.createElement("div");
        content.style.cssText = "flex:1;overflow:auto;padding:16px";
        content.innerHTML = "<div style='text-align:center;padding:40px'><img src='/stage/loading-pure.svg' style='width:48px'></div>";
        
        container.appendChild(toolbar);
        container.appendChild(content);
        this.element.appendChild(container);

        try {
            const response = await fetch(fileUrl);
            if (!response.ok) throw new Error("HTTP " + response.status);
            const text = await response.text();
            
            if (ext === ".md" || ext === ".markdown") {
                // Markdown 文件显示为格式化的代码
                const pre = document.createElement("pre");
                pre.style.cssText = "margin:0;padding:16px;background:var(--b3-theme-surface);border-radius:4px;overflow:auto;font-family:var(--b3-font-family-code);font-size:13px;line-height:1.6;white-space:pre-wrap;word-break:break-all";
                const code = document.createElement("code");
                code.textContent = text;
                pre.appendChild(code);
                content.innerHTML = "";
                const wrapper = document.createElement("div");
                wrapper.style.cssText = "max-width:900px;margin:0 auto";
                wrapper.appendChild(pre);
                content.appendChild(wrapper);
            } else {
                const pre = document.createElement("pre");
                pre.style.cssText = "margin:0;padding:16px;background:var(--b3-theme-surface);border-radius:4px;overflow:auto;font-family:var(--b3-font-family-code);font-size:13px;line-height:1.6;white-space:pre-wrap;word-break:break-all";
                const code = document.createElement("code");
                code.textContent = text;
                pre.appendChild(code);
                content.innerHTML = "";
                content.appendChild(pre);
            }
        } catch (e) {
            content.innerHTML = "<div style='text-align:center;color:var(--b3-theme-error)'>加载失败: " + e.message + "</div>";
        }
    }


    private async renderOfficeFile(localUrl: string) {
        const fileName = this.getFileName();
        const ext = pathPosix().extname(this.path).toLowerCase().split("?")[0];
        
        this.element.innerHTML = "";
        const container = document.createElement("div");
        container.style.cssText = "width:100%;height:100%;display:flex;flex-direction:column;background:var(--b3-theme-background)";
        
        const toolbar = document.createElement("div");
        toolbar.style.cssText = "padding:8px 16px;background:var(--b3-theme-surface);border-bottom:1px solid var(--b3-border-color);display:flex;align-items:center;gap:8px";
        
        const title = document.createElement("span");
        title.style.cssText = "color:var(--b3-theme-on-surface);font-size:13px;font-weight:500";
        title.textContent = fileName;
        toolbar.appendChild(title);
        
        const spacer = document.createElement("div");
        spacer.style.flex = "1";
        toolbar.appendChild(spacer);
        
        const downloadBtn = document.createElement("a");
        downloadBtn.href = localUrl;
        downloadBtn.download = "";
        downloadBtn.className = "b3-button b3-button--small b3-button--outline";
        downloadBtn.style.textDecoration = "none";
        downloadBtn.textContent = "下载";
        toolbar.appendChild(downloadBtn);
        
        const content = document.createElement("div");
        content.style.cssText = "flex:1;overflow:auto;padding:16px";
        content.innerHTML = "<div style='text-align:center;padding:40px'><img src='/stage/loading-pure.svg' style='width:48px'><p style='margin-top:16px;color:var(--b3-theme-on-surface-light)'>正在加载文档...</p></div>";
        
        container.appendChild(toolbar);
        container.appendChild(content);
        this.element.appendChild(container);
        
        // 对于 .docx 文件，使用 mammoth.js 本地预览
        if (ext === ".docx") {
            try {
                const response = await fetch(localUrl);
                if (!response.ok) throw new Error("HTTP " + response.status);
                const arrayBuffer = await response.arrayBuffer();
                
                const result = await mammoth.convertToHtml({ arrayBuffer });
                
                const wrapper = document.createElement("div");
                wrapper.style.cssText = "max-width:900px;margin:0 auto;background:white;padding:32px;border-radius:4px;box-shadow:0 1px 3px rgba(0,0,0,0.1)";
                wrapper.innerHTML = result.value;
                
                // 添加基本样式
                const style = document.createElement("style");
                style.textContent = `
                    .docx-preview { color: #333; font-family: 'Times New Roman', serif; line-height: 1.6; }
                    .docx-preview h1 { font-size: 24px; margin: 24px 0 16px; }
                    .docx-preview h2 { font-size: 20px; margin: 20px 0 12px; }
                    .docx-preview h3 { font-size: 16px; margin: 16px 0 8px; }
                    .docx-preview p { margin: 8px 0; }
                    .docx-preview table { border-collapse: collapse; width: 100%; margin: 16px 0; }
                    .docx-preview td, .docx-preview th { border: 1px solid #ddd; padding: 8px; }
                    .docx-preview img { max-width: 100%; height: auto; }
                    .docx-preview ul, .docx-preview ol { margin: 8px 0; padding-left: 24px; }
                `;
                wrapper.classList.add("docx-preview");
                
                content.innerHTML = "";
                content.appendChild(style);
                content.appendChild(wrapper);
                
                // 显示警告信息（如果有）
                if (result.messages && result.messages.length > 0) {
                    console.log("[DocumentViewer] mammoth warnings:", result.messages);
                }
            } catch (e) {
                console.error("[DocumentViewer] mammoth error:", e);
                content.innerHTML = `<div style='text-align:center;padding:40px'>
                    <p style='color:var(--b3-theme-error);margin-bottom:16px'>文档预览失败: ${e.message}</p>
                    <a href='${localUrl}' download class='b3-button b3-button--outline' style='text-decoration:none'>下载文件</a>
                </div>`;
            }
        } else {
            // 其他 Office 文件类型显示下载提示
            content.innerHTML = `<div style='text-align:center;padding:40px'>
                <p style='color:var(--b3-theme-on-surface-light);margin-bottom:16px'>此文件类型 (${ext}) 暂不支持在线预览</p>
                <a href='${localUrl}' download class='b3-button b3-button--outline' style='text-decoration:none'>下载文件</a>
            </div>`;
        }
    }

    private renderGenericFile(fileUrl: string) {
        const fileName = this.getFileName();
        this.element.innerHTML = "";
        
        const container = document.createElement("div");
        container.style.cssText = "width:100%;height:100%;display:flex;flex-direction:column;align-items:center;justify-content:center;background:var(--b3-theme-background)";
        
        const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
        icon.style.cssText = "width:80px;height:80px;color:var(--b3-theme-on-surface-light)";
        icon.innerHTML = "<use xlink:href='#iconFile'></use>";
        container.appendChild(icon);
        
        const h3 = document.createElement("h3");
        h3.style.cssText = "margin:24px 0 8px;color:var(--b3-theme-on-background);font-size:18px";
        h3.textContent = fileName;
        container.appendChild(h3);
        
        const p = document.createElement("p");
        p.style.cssText = "margin:0 0 24px;color:var(--b3-theme-on-surface-light);font-size:13px";
        p.textContent = "此文件类型暂不支持预览";
        container.appendChild(p);
        
        const downloadBtn = document.createElement("a");
        downloadBtn.href = fileUrl;
        downloadBtn.download = "";
        downloadBtn.className = "b3-button b3-button--outline";
        downloadBtn.style.textDecoration = "none";
        downloadBtn.textContent = "下载文件";
        container.appendChild(downloadBtn);
        
        this.element.appendChild(container);
    }

    private getFileName(): string {
        try {
            return decodeURIComponent(pathPosix().basename(this.path).split("?")[0]);
        } catch {
            return pathPosix().basename(this.path).split("?")[0];
        }
    }
}
