import {Model} from "../layout/Model";
import {Tab} from "../layout/Tab";
import {setPanelFocus} from "../layout/util";
import {App} from "../index";
import {clearOBG} from "../layout/dock/util";
import {pathPosix} from "../util/pathName";
import * as mammoth from "mammoth";
import JSZip from "jszip";
import * as XLSX from "xlsx";

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
        } else if (ext === ".pptx") {
            // 对于 .pptx 文件，使用 JSZip 解析并渲染
            await this.renderPptxFile(localUrl, content);
        } else if (ext === ".xls" || ext === ".xlsx") {
            // 对于 Excel 文件，使用 xlsx 库解析并渲染
            await this.renderExcelFile(localUrl, content);
        } else {
            // 其他 Office 文件类型显示下载提示
            content.innerHTML = `<div style='text-align:center;padding:40px'>
                <p style='color:var(--b3-theme-on-surface-light);margin-bottom:16px'>此文件类型 (${ext}) 暂不支持在线预览</p>
                <a href='${localUrl}' download class='b3-button b3-button--outline' style='text-decoration:none'>下载文件</a>
            </div>`;
        }
    }

    private async renderPptxFile(localUrl: string, content: HTMLElement) {
        try {
            const response = await fetch(localUrl);
            if (!response.ok) throw new Error("HTTP " + response.status);
            const arrayBuffer = await response.arrayBuffer();
            
            const zip = await JSZip.loadAsync(arrayBuffer);
            
            // 获取幻灯片数量
            const slideFiles: string[] = [];
            zip.forEach((relativePath) => {
                if (relativePath.match(/^ppt\/slides\/slide\d+\.xml$/)) {
                    slideFiles.push(relativePath);
                }
            });
            
            // 按数字排序
            slideFiles.sort((a, b) => {
                const numA = parseInt(a.match(/slide(\d+)\.xml/)?.[1] || "0");
                const numB = parseInt(b.match(/slide(\d+)\.xml/)?.[1] || "0");
                return numA - numB;
            });
            
            if (slideFiles.length === 0) {
                throw new Error("无法解析 PPTX 文件：未找到幻灯片");
            }
            
            // 解析幻灯片内容
            const slides: { texts: string[], images: string[] }[] = [];
            const mediaFiles: Map<string, string> = new Map();
            
            // 提取媒体文件
            const mediaPromises: Promise<void>[] = [];
            zip.forEach((relativePath, file) => {
                if (relativePath.startsWith("ppt/media/")) {
                    const promise = file.async("base64").then(base64 => {
                        const ext = relativePath.split(".").pop()?.toLowerCase() || "png";
                        const mimeType = ext === "jpg" || ext === "jpeg" ? "image/jpeg" : 
                                        ext === "png" ? "image/png" : 
                                        ext === "gif" ? "image/gif" : "image/png";
                        mediaFiles.set(relativePath, `data:${mimeType};base64,${base64}`);
                    });
                    mediaPromises.push(promise);
                }
            });
            await Promise.all(mediaPromises);
            
            // 解析每张幻灯片
            for (const slideFile of slideFiles) {
                const slideXml = await zip.file(slideFile)?.async("string");
                if (!slideXml) continue;
                
                const slideData: { texts: string[], images: string[] } = { texts: [], images: [] };
                
                // 提取文本内容
                const textMatches = slideXml.matchAll(/<a:t>([^<]*)<\/a:t>/g);
                for (const match of textMatches) {
                    const text = match[1].trim();
                    if (text) {
                        slideData.texts.push(text);
                    }
                }
                
                // 解析关系文件获取图片引用
                const slideNum = slideFile.match(/slide(\d+)\.xml/)?.[1];
                const relsFile = `ppt/slides/_rels/slide${slideNum}.xml.rels`;
                const relsXml = await zip.file(relsFile)?.async("string");
                
                if (relsXml) {
                    const relMatches = relsXml.matchAll(/Target="([^"]*media\/[^"]*)"/g);
                    for (const match of relMatches) {
                        let target = match[1];
                        if (target.startsWith("../")) {
                            target = "ppt/" + target.substring(3);
                        }
                        const dataUrl = mediaFiles.get(target);
                        if (dataUrl) {
                            slideData.images.push(dataUrl);
                        }
                    }
                }
                
                slides.push(slideData);
            }
            
            // 渲染幻灯片
            content.innerHTML = "";
            
            const style = document.createElement("style");
            style.textContent = `
                .pptx-container { max-width: 1000px; margin: 0 auto; }
                .pptx-slide { 
                    background: white; 
                    border-radius: 8px; 
                    box-shadow: 0 2px 8px rgba(0,0,0,0.15); 
                    margin-bottom: 24px; 
                    overflow: hidden;
                }
                .pptx-slide-header {
                    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                    color: white;
                    padding: 8px 16px;
                    font-size: 12px;
                    font-weight: 500;
                }
                .pptx-slide-content {
                    padding: 24px;
                    min-height: 200px;
                    aspect-ratio: 16/9;
                    display: flex;
                    flex-direction: column;
                    justify-content: center;
                }
                .pptx-slide-text {
                    color: #333;
                    font-size: 18px;
                    line-height: 1.6;
                    margin-bottom: 8px;
                }
                .pptx-slide-text:first-child {
                    font-size: 24px;
                    font-weight: 600;
                    color: #1a1a1a;
                    margin-bottom: 16px;
                }
                .pptx-slide-images {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 12px;
                    margin-top: 16px;
                }
                .pptx-slide-images img {
                    max-width: 100%;
                    max-height: 300px;
                    border-radius: 4px;
                    object-fit: contain;
                }
                .pptx-nav {
                    position: sticky;
                    top: 0;
                    background: var(--b3-theme-surface);
                    padding: 12px 16px;
                    margin: -16px -16px 16px -16px;
                    border-bottom: 1px solid var(--b3-border-color);
                    display: flex;
                    align-items: center;
                    gap: 12px;
                    z-index: 10;
                }
                .pptx-nav-info {
                    color: var(--b3-theme-on-surface);
                    font-size: 13px;
                }
            `;
            content.appendChild(style);
            
            const nav = document.createElement("div");
            nav.className = "pptx-nav";
            nav.innerHTML = `<span class="pptx-nav-info">共 ${slides.length} 张幻灯片</span>`;
            content.appendChild(nav);
            
            const container = document.createElement("div");
            container.className = "pptx-container";
            
            slides.forEach((slide, index) => {
                const slideEl = document.createElement("div");
                slideEl.className = "pptx-slide";
                slideEl.id = `slide-${index + 1}`;
                
                const header = document.createElement("div");
                header.className = "pptx-slide-header";
                header.textContent = `幻灯片 ${index + 1}`;
                slideEl.appendChild(header);
                
                const contentDiv = document.createElement("div");
                contentDiv.className = "pptx-slide-content";
                
                // 添加文本
                slide.texts.forEach(text => {
                    const p = document.createElement("p");
                    p.className = "pptx-slide-text";
                    p.textContent = text;
                    contentDiv.appendChild(p);
                });
                
                // 添加图片
                if (slide.images.length > 0) {
                    const imagesDiv = document.createElement("div");
                    imagesDiv.className = "pptx-slide-images";
                    slide.images.forEach(src => {
                        const img = document.createElement("img");
                        img.src = src;
                        img.alt = "幻灯片图片";
                        imagesDiv.appendChild(img);
                    });
                    contentDiv.appendChild(imagesDiv);
                }
                
                // 如果没有内容，显示空白提示
                if (slide.texts.length === 0 && slide.images.length === 0) {
                    const empty = document.createElement("p");
                    empty.style.cssText = "color: #999; font-style: italic; text-align: center;";
                    empty.textContent = "(空白幻灯片)";
                    contentDiv.appendChild(empty);
                }
                
                slideEl.appendChild(contentDiv);
                container.appendChild(slideEl);
            });
            
            content.appendChild(container);
            
        } catch (e) {
            console.error("[DocumentViewer] PPTX preview error:", e);
            content.innerHTML = `<div style='text-align:center;padding:40px'>
                <p style='color:var(--b3-theme-error);margin-bottom:16px'>PPT 预览失败: ${e.message}</p>
                <a href='${localUrl}' download class='b3-button b3-button--outline' style='text-decoration:none'>下载文件</a>
            </div>`;
        }
    }

    private async renderExcelFile(localUrl: string, content: HTMLElement) {
        try {
            const response = await fetch(localUrl);
            if (!response.ok) throw new Error("HTTP " + response.status);
            const arrayBuffer = await response.arrayBuffer();
            
            // 使用 xlsx 库解析 Excel 文件
            const workbook = XLSX.read(arrayBuffer, { type: "array" });
            
            if (!workbook.SheetNames || workbook.SheetNames.length === 0) {
                throw new Error("无法解析 Excel 文件：未找到工作表");
            }
            
            content.innerHTML = "";
            
            // 添加样式
            const style = document.createElement("style");
            style.textContent = `
                .xlsx-container { max-width: 100%; margin: 0 auto; }
                .xlsx-tabs {
                    display: flex;
                    gap: 4px;
                    padding: 8px 0;
                    border-bottom: 1px solid var(--b3-border-color);
                    margin-bottom: 16px;
                    flex-wrap: wrap;
                }
                .xlsx-tab {
                    padding: 6px 16px;
                    background: var(--b3-theme-surface);
                    border: 1px solid var(--b3-border-color);
                    border-radius: 4px;
                    cursor: pointer;
                    font-size: 13px;
                    color: var(--b3-theme-on-surface);
                    transition: all 0.2s;
                }
                .xlsx-tab:hover {
                    background: var(--b3-theme-surface-lighter);
                }
                .xlsx-tab.active {
                    background: var(--b3-theme-primary);
                    color: var(--b3-theme-on-primary);
                    border-color: var(--b3-theme-primary);
                }
                .xlsx-sheet {
                    display: none;
                    overflow: auto;
                }
                .xlsx-sheet.active {
                    display: block;
                }
                .xlsx-table {
                    border-collapse: collapse;
                    width: 100%;
                    font-size: 13px;
                    background: white;
                }
                .xlsx-table th, .xlsx-table td {
                    border: 1px solid #ddd;
                    padding: 8px 12px;
                    text-align: left;
                    white-space: nowrap;
                    max-width: 300px;
                    overflow: hidden;
                    text-overflow: ellipsis;
                }
                .xlsx-table th {
                    background: #f5f5f5;
                    font-weight: 600;
                    position: sticky;
                    top: 0;
                    z-index: 1;
                }
                .xlsx-table tr:nth-child(even) {
                    background: #fafafa;
                }
                .xlsx-table tr:hover {
                    background: #f0f7ff;
                }
                .xlsx-row-num {
                    background: #f5f5f5 !important;
                    color: #666;
                    font-weight: 500;
                    text-align: center !important;
                    min-width: 40px;
                }
                .xlsx-info {
                    color: var(--b3-theme-on-surface-light);
                    font-size: 12px;
                    margin-top: 8px;
                }
            `;
            content.appendChild(style);
            
            // 创建标签页
            const tabsContainer = document.createElement("div");
            tabsContainer.className = "xlsx-tabs";
            
            const sheetsContainer = document.createElement("div");
            sheetsContainer.className = "xlsx-container";
            
            workbook.SheetNames.forEach((sheetName, index) => {
                // 创建标签
                const tab = document.createElement("button");
                tab.className = "xlsx-tab" + (index === 0 ? " active" : "");
                tab.textContent = sheetName;
                tab.onclick = () => {
                    // 切换标签
                    tabsContainer.querySelectorAll(".xlsx-tab").forEach(t => t.classList.remove("active"));
                    tab.classList.add("active");
                    sheetsContainer.querySelectorAll(".xlsx-sheet").forEach(s => s.classList.remove("active"));
                    sheetsContainer.querySelector(`#sheet-${index}`)?.classList.add("active");
                };
                tabsContainer.appendChild(tab);
                
                // 创建工作表内容
                const sheetDiv = document.createElement("div");
                sheetDiv.className = "xlsx-sheet" + (index === 0 ? " active" : "");
                sheetDiv.id = `sheet-${index}`;
                
                const sheet = workbook.Sheets[sheetName];
                const range = XLSX.utils.decode_range(sheet["!ref"] || "A1");
                const rowCount = range.e.r - range.s.r + 1;
                const colCount = range.e.c - range.s.c + 1;
                
                // 限制显示的行数和列数，避免性能问题
                const maxRows = Math.min(rowCount, 1000);
                const maxCols = Math.min(colCount, 50);
                
                const table = document.createElement("table");
                table.className = "xlsx-table";
                
                // 创建表头（列字母）
                const thead = document.createElement("thead");
                const headerRow = document.createElement("tr");
                
                // 行号列
                const cornerTh = document.createElement("th");
                cornerTh.className = "xlsx-row-num";
                cornerTh.textContent = "";
                headerRow.appendChild(cornerTh);
                
                // 列字母
                for (let c = 0; c < maxCols; c++) {
                    const th = document.createElement("th");
                    th.textContent = XLSX.utils.encode_col(range.s.c + c);
                    headerRow.appendChild(th);
                }
                thead.appendChild(headerRow);
                table.appendChild(thead);
                
                // 创建表体
                const tbody = document.createElement("tbody");
                for (let r = 0; r < maxRows; r++) {
                    const tr = document.createElement("tr");
                    
                    // 行号
                    const rowNumTd = document.createElement("td");
                    rowNumTd.className = "xlsx-row-num";
                    rowNumTd.textContent = String(range.s.r + r + 1);
                    tr.appendChild(rowNumTd);
                    
                    // 单元格数据
                    for (let c = 0; c < maxCols; c++) {
                        const td = document.createElement("td");
                        const cellAddress = XLSX.utils.encode_cell({ r: range.s.r + r, c: range.s.c + c });
                        const cell = sheet[cellAddress];
                        if (cell) {
                            // 格式化显示值
                            if (cell.t === "n" && cell.v !== undefined) {
                                // 数字类型
                                td.textContent = cell.w || String(cell.v);
                                td.style.textAlign = "right";
                            } else if (cell.t === "d") {
                                // 日期类型
                                td.textContent = cell.w || String(cell.v);
                            } else {
                                td.textContent = cell.w || cell.v?.toString() || "";
                            }
                            td.title = td.textContent; // 鼠标悬停显示完整内容
                        }
                        tr.appendChild(td);
                    }
                    tbody.appendChild(tr);
                }
                table.appendChild(tbody);
                sheetDiv.appendChild(table);
                
                // 显示信息
                const info = document.createElement("div");
                info.className = "xlsx-info";
                let infoText = `共 ${rowCount} 行 × ${colCount} 列`;
                if (rowCount > maxRows || colCount > maxCols) {
                    infoText += ` (显示前 ${maxRows} 行 × ${maxCols} 列)`;
                }
                info.textContent = infoText;
                sheetDiv.appendChild(info);
                
                sheetsContainer.appendChild(sheetDiv);
            });
            
            content.appendChild(tabsContainer);
            content.appendChild(sheetsContainer);
            
        } catch (e) {
            console.error("[DocumentViewer] Excel preview error:", e);
            content.innerHTML = `<div style='text-align:center;padding:40px'>
                <p style='color:var(--b3-theme-error);margin-bottom:16px'>Excel 预览失败: ${e.message}</p>
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
