import { isPaidUser, needSubscribe } from "../util/needSubscribe";
import { showMessage } from "../dialog/message";
import { fetchPost } from "../util/fetch";
import { Dialog } from "../dialog";
import { confirmDialog } from "../dialog/confirmDialog";
import { isMobile } from "../util/functions";
import { processSync } from "../dialog/processSystem";
/// #if !MOBILE
import { openSetting } from "../config";
/// #endif
import { App } from "../index";
import { Constants } from "../constants";
import { getCloudURL } from "../config/util/about";

export const addCloudName = (cloudPanelElement: Element) => {
    const dialog = new Dialog({
        title: window.siyuan.languages.cloudSyncDir,
        content: `<div class="b3-dialog__content">
    <input class="b3-text-field fn__block" value="main">
    <div class="b3-label__text">${window.siyuan.languages.reposTip}</div>
</div>
<div class="b3-dialog__action">
    <button class="b3-button b3-button--cancel">${window.siyuan.languages.cancel}</button><div class="fn__space"></div>
    <button class="b3-button b3-button--text">${window.siyuan.languages.confirm}</button>
</div>`,
        width: isMobile() ? "92vw" : "520px",
    });
    dialog.element.setAttribute("data-key", Constants.DIALOG_SYNCADDCLOUDDIR);
    const inputElement = dialog.element.querySelector("input") as HTMLInputElement;
    const btnsElement = dialog.element.querySelectorAll(".b3-button");
    dialog.bindInput(inputElement, () => {
        (btnsElement[1] as HTMLButtonElement).click();
    });
    inputElement.focus();
    inputElement.select();
    btnsElement[0].addEventListener("click", () => {
        dialog.destroy();
    });
    btnsElement[1].addEventListener("click", () => {
        cloudPanelElement.innerHTML = '<img style="margin: 0 auto;display: block;width: 64px;height: 100%" src="/stage/loading-pure.svg">';
        fetchPost("/api/sync/createCloudSyncDir", { name: inputElement.value }, () => {
            dialog.destroy();
            getSyncCloudList(cloudPanelElement, true);
        });
    });
};

export const bindSyncCloudListEvent = (cloudPanelElement: Element, cb?: () => void) => {
    cloudPanelElement.addEventListener("click", (event) => {
        let target = event.target as HTMLElement;
        while (target && !target.isEqualNode(cloudPanelElement)) {
            const type = target.getAttribute("data-type");
            if (type) {
                switch (type) {
                    case "addCloud":
                        addCloudName(cloudPanelElement);
                        break;
                    case "removeCloud":
                        confirmDialog(window.siyuan.languages.deleteOpConfirm, `${window.siyuan.languages.confirmDeleteCloudDir} <i>${target.parentElement.getAttribute("data-name")}</i>`, () => {
                            cloudPanelElement.innerHTML = '<img style="margin: 0 auto;display: block;width: 64px;height: 100%" src="/stage/loading-pure.svg">';
                            fetchPost("/api/sync/removeCloudSyncDir", { name: target.parentElement.getAttribute("data-name") }, (response) => {
                                window.siyuan.config.sync.cloudName = response.data;
                                getSyncCloudList(cloudPanelElement, true, cb);
                            });
                        }, undefined, true);
                        break;
                    case "selectCloud":
                        cloudPanelElement.innerHTML = '<img style="margin: 0 auto;display: block;width: 64px;height: 100%" src="/stage/loading-pure.svg">';
                        fetchPost("/api/sync/setCloudSyncDir", { name: target.getAttribute("data-name") }, () => {
                            window.siyuan.config.sync.cloudName = target.getAttribute("data-name");
                            getSyncCloudList(cloudPanelElement, true, cb);
                        });
                        break;
                }
                event.preventDefault();
                event.stopPropagation();
                break;
            }
            target = target.parentElement;
        }
    });
};

export const getSyncCloudList = (cloudPanelElement: Element, reload = false, cb?: () => void) => {
    if (!reload && cloudPanelElement.firstElementChild.tagName !== "IMG") {
        return;
    }
    fetchPost("/api/sync/listCloudSyncDir", {}, (response) => {
        let syncListHTML = `<ul><li style="padding: 0 16px" class="b3-list--empty">${window.siyuan.languages.emptyCloudSyncList}</li></ul>`;
        if (response.code === 1) {
            syncListHTML = `<ul>
    <li class="b3-list--empty ft__error">
        ${response.msg}
    </li>
    <li class="b3-list--empty">
        ${window.siyuan.languages.cloudConfigTip}
    </li>
</ul>`;
        } else if (response.code !== 1) {
            syncListHTML = '<ul class="b3-list b3-list--background fn__flex-1" style="overflow: auto;">';
            response.data.syncDirs.forEach((item: { hSize: string, cloudName: string, updated: string }) => {
                /// #if MOBILE
                syncListHTML += `<li data-type="selectCloud" data-name="${item.cloudName}" class="b3-list-item b3-list-item--two">
    <div class="b3-list-item__first" data-name="${item.cloudName}">
        <input type="radio" name="cloudName"${item.cloudName === response.data.checkedSyncDir ? " checked" : ""}/>
        <span class="fn__space"></span>
        <span>${item.cloudName}</span>
        <span class="fn__flex-1 fn__space"></span>
        <span data-type="removeCloud" class="b3-list-item__action">
            <svg><use xlink:href="#iconTrashcan"></use></svg>
        </span>
    </div>
    <div class="b3-list-item__meta fn__flex">
        <span>${item.hSize}</span>
        <span class="fn__flex-1 fn__space"></span>
        <span>${item.updated}</span>
    </div>
</li>`;
                /// #else
                syncListHTML += `<li data-type="selectCloud" data-name="${item.cloudName}" class="b3-list-item b3-list-item--narrow b3-list-item--hide-action">
<input type="radio" name="cloudName"${item.cloudName === response.data.checkedSyncDir ? " checked" : ""}/>
<span class="fn__space"></span>
<span>${item.cloudName}</span>
<span class="fn__space"></span>
<span class="ft__on-surface">${item.hSize}</span>
<span class="b3-list-item__meta">${item.updated}</span>
<span class="fn__flex-1 fn__space"></span>
<span data-type="removeCloud" class="b3-tooltips b3-tooltips__w b3-list-item__action${(window.siyuan.config.sync.provider === 2 || window.siyuan.config.sync.provider === 3) ? " fn__none" : ""}" aria-label="${window.siyuan.languages.delete}">
    <svg><use xlink:href="#iconTrashcan"></use></svg>
</span></li>`;
                /// #endif
            });
            syncListHTML += `</ul>
<div class="fn__hr"></div>
<div class="fn__flex">
    <div class="fn__flex-1"></div>
    <button class="b3-button b3-button--outline${(window.siyuan.config.sync.provider === 2 || window.siyuan.config.sync.provider === 3) ? " fn__none" : ""}" data-type="addCloud"><svg><use xlink:href="#iconAdd"></use></svg>${window.siyuan.languages.addAttr}</button>
</div>`;
        }
        cloudPanelElement.innerHTML = syncListHTML;
        if (cb) {
            cb();
        }
    });
};

export const syncGuide = (app?: App) => {
    if (window.siyuan.config.readonly) {
        return;
    }
    /// #if MOBILE
    if (0 === window.siyuan.config.sync.provider) {
        if (needSubscribe()) {
            return;
        }
    } else if (!isPaidUser()) {
        showMessage(window.siyuan.languages["_kernel"][214].replaceAll("${accountServer}", getCloudURL("")));
        return;
    }
    /// #else
    if (document.querySelector("#barSync")?.classList.contains("toolbar__item--active")) {
        return;
    }
    if (0 === window.siyuan.config.sync.provider && needSubscribe("") && app) {
        const dialogSetting = openSetting(app);
        if (window.siyuan.user) {
            dialogSetting.element.querySelector('.b3-tab-bar [data-name="repos"]').dispatchEvent(new CustomEvent("click"));
        } else {
            dialogSetting.element.querySelector('.b3-tab-bar [data-name="account"]').dispatchEvent(new CustomEvent("click"));
            dialogSetting.element.querySelector('.config__tab-container[data-name="account"]').setAttribute("data-action", "go-repos");
        }
        return;
    }
    if (0 !== window.siyuan.config.sync.provider && !isPaidUser() && app) {
        showMessage(window.siyuan.languages["_kernel"][214].replaceAll("${accountServer}", getCloudURL("")));
        return;
    }
    /// #endif

    // 🔥 简化流程：移除密码设置检查，直接进入同步
    if (!window.siyuan.config.repo.key) {
        // 自动生成一个默认密钥，无需用户输入密码
        autoInitKey();
        return;
    }

    if (!window.siyuan.config.sync.enabled) {
        setSync();
        return;
    }
    syncNow();
};

// 🆕 自动初始化密钥（无需用户输入密码）
const autoInitKey = () => {
    // 使用设备ID和时间戳生成唯一密钥
    const deviceKey = window.siyuan.config.system.id || 'default-device';
    const autoPass = `auto-${deviceKey}-${Date.now()}`;

    fetchPost("/api/repo/initRepoKeyFromPassphrase", { pass: autoPass }, (response) => {
        window.siyuan.config.repo.key = response.data.key;
        showMessage("✅ 已自动生成同步密钥", 2000, "info");

        // 继续同步流程
        if (!window.siyuan.config.sync.enabled) {
            setSync();
        } else {
            syncNow();
        }
    });
};

const syncNow = () => {
    // 🔥 简化：默认使用智能合并模式
    if (window.siyuan.config.sync.mode !== 3) {
        // 添加合并模式提示
        confirmDialog(
            "🔄 开始同步",
            `<div class="b3-dialog__content">
                <div class="ft__on-surface" style="margin-bottom: 12px;">
                    💡 使用<strong>智能合并模式</strong>，会自动合并本地和云端数据，避免内容丢失。
                </div>
                <div class="ft__secondary" style="font-size: 12px; line-height: 1.6;">
                    • 优先保留较新的修改<br>
                    • 发生冲突时会生成冲突文档<br>
                    • 不会删除任何现有内容
                </div>
            </div>`,
            () => {
                fetchPost("/api/sync/performSync", { merge: true });
            },
            () => {
                // 取消同步
            }
        );
        return;
    }

    // 完全手动模式：提供更多选项
    const manualDialog = new Dialog({
        title: "🔄 选择同步方式",
        content: `<div class="b3-dialog__content">
    <label class="fn__flex b3-label" style="margin-bottom: 16px;">
        <input type="radio" name="syncMode" value="merge" checked>
        <span class="fn__space"></span>
        <div>
            <div style="font-weight: 500;">🔀 智能合并（推荐）</div>
            <div class="b3-label__text">
                自动合并本地和云端数据，优先保留较新修改，冲突时生成冲突文档
            </div>
        </div>
    </label>
    <label class="fn__flex b3-label" style="margin-bottom: 16px;">
        <input type="radio" name="syncMode" value="upload">
        <span class="fn__space"></span>
        <div>
            <div style="font-weight: 500;">⬆️ 上传到云端</div>
            <div class="b3-label__text">
                ${window.siyuan.languages.uploadData2CloudTip}
            </div>
        </div>
    </label>
    <label class="fn__flex b3-label">
        <input type="radio" name="syncMode" value="download">
        <span class="fn__space"></span>
        <div>
            <div style="font-weight: 500;">⬇️ 从云端下载</div>
            <div class="b3-label__text">
                ${window.siyuan.languages.downloadDataFromCloudTip}
            </div>
        </div>
    </label>
    <div class="fn__hr"></div>
    <div style="background: var(--b3-theme-surface-lighter); padding: 12px; border-radius: 4px; font-size: 12px; line-height: 1.6;">
        💡 <strong>提示</strong>：首次同步建议选择"智能合并"，系统会自动处理数据合并，确保不丢失内容。
    </div>
</div>
<div class="b3-dialog__action">
    <button class="b3-button b3-button--cancel">${window.siyuan.languages.cancel}</button><div class="fn__space"></div>
    <button class="b3-button b3-button--text">开始同步</button>
</div>`,
        width: isMobile() ? "92vw" : "560px",
    });
    manualDialog.element.setAttribute("data-key", Constants.DIALOG_SYNCCHOOSEDIRECTION);
    const btnsElement = manualDialog.element.querySelectorAll(".b3-button");
    btnsElement[0].addEventListener("click", () => {
        manualDialog.destroy();
    });
    btnsElement[1].addEventListener("click", () => {
        const modeElement = manualDialog.element.querySelector("input[name=syncMode]:checked") as HTMLInputElement;
        if (!modeElement) {
            showMessage("请选择同步方式");
            return;
        }

        const mode = modeElement.value;
        if (mode === "merge") {
            // 智能合并模式
            fetchPost("/api/sync/performSync", { merge: true });
        } else if (mode === "upload") {
            // 上传模式
            fetchPost("/api/sync/performSync", { upload: true });
        } else {
            // 下载模式
            fetchPost("/api/sync/performSync", { upload: false });
        }
        manualDialog.destroy();
    });
};

const setSync = (key?: string, dialog?: Dialog) => {
    if (key) {
        window.siyuan.config.repo.key = key;
    }
    if (!window.siyuan.config.sync.enabled) {
        const listHTML = `<div class="b3-dialog__content">
    <div class="ft__on-surface">${window.siyuan.languages.syncConfGuide3}</div>
    <div class="fn__hr--b"></div>
    <div style="display: flex;flex-direction: column;height: 40vh;">
        <img style="margin: 0 auto;display: block;width: 64px;height: 100%" src="/stage/loading-pure.svg">
    </div>
</div>
<div class="b3-dialog__action">
    <button class="b3-button" disabled="disabled">${window.siyuan.languages.openSyncTip1}</button>
</div>`;
        if (dialog) {
            dialog.element.querySelector(".b3-dialog__header").innerHTML = "🗂️ " + window.siyuan.languages.cloudSyncDir;
            dialog.element.querySelector(".b3-dialog__body").innerHTML = listHTML;
        } else {
            dialog = new Dialog({
                title: "🗂️ " + window.siyuan.languages.cloudSyncDir,
                content: listHTML,
                width: isMobile() ? "92vw" : "520px",
            });
        }
        dialog.element.setAttribute("data-key", Constants.DIALOG_SYNCCHOOSEDIR);
        const contentElement = dialog.element.querySelector(".b3-dialog__content").lastElementChild;
        const btnElement = dialog.element.querySelector(".b3-button");
        bindSyncCloudListEvent(contentElement, () => {
            if (contentElement.querySelector("input[checked]")) {
                btnElement.removeAttribute("disabled");
            } else {
                btnElement.setAttribute("disabled", "disabled");
            }
        });
        getSyncCloudList(contentElement, false, () => {
            if (contentElement.querySelector("input[checked]")) {
                btnElement.removeAttribute("disabled");
            } else {
                btnElement.setAttribute("disabled", "disabled");
            }
        });
        btnElement.addEventListener("click", () => {
            dialog.destroy();
            fetchPost("/api/sync/setSyncEnable", { enabled: true }, () => {
                window.siyuan.config.sync.enabled = true;
                processSync();
                confirmDialog("🔄 " + window.siyuan.languages.syncConfGuide4, window.siyuan.languages.syncConfGuide5, () => {
                    syncNow();
                });
            });
        });
    } else {
        if (dialog) {
            dialog.destroy();
        }
        confirmDialog("🔄 " + window.siyuan.languages.syncConfGuide4, window.siyuan.languages.syncConfGuide5, () => {
            syncNow();
        });
    }
};

// 🔥 保留但简化 setKey 函数，仅供手动设置密码使用（可选）
export const setKey = (isSync: boolean, cb?: () => void) => {
    // 现在默认自动生成密钥，此函数仅在用户手动要求设置密码时调用
    confirmDialog(
        "🔑 同步密钥设置",
        `<div class="b3-dialog__content">
            <div class="ft__on-surface" style="margin-bottom: 12px;">
                系统已为您自动生成同步密钥，无需手动设置密码。
            </div>
            <div class="ft__secondary" style="font-size: 12px; line-height: 1.6;">
                💡 自动生成的密钥已足够安全<br>
                💡 如需自定义密码，请前往设置页面
            </div>
        </div>`,
        () => {
            // 自动初始化
            autoInitKey();
        },
        () => {
            // 取消
        }
    );
};
