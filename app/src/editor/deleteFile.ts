import {fetchPost} from "../util/fetch";
import {getDisplayName, getNotebookName} from "../util/pathName";
import {confirmDialog} from "../dialog/confirmDialog";
import {hasTopClosestByTag} from "../protyle/util/hasClosest";
import {Constants} from "../constants";
import {showMessage} from "../dialog/message";
import {escapeHtml} from "../util/escape";
import {getDockByType} from "../layout/tabUtil";
import {Files} from "../layout/dock/Files";

// 从文档树中移除指定的文档元素
const removeDocFromTree = (pathString: string) => {
    const fileModel = getDockByType("file")?.data?.file;
    if (fileModel instanceof Files) {
        const targetElement = fileModel.element.querySelector(`li[data-path="${pathString}"]`);
        if (targetElement) {
            // 如果有子节点列表，也一并删除
            if (targetElement.nextElementSibling?.tagName === "UL") {
                targetElement.nextElementSibling.remove();
            }
            // 检查父节点是否还有其他子节点
            const parentUL = targetElement.parentElement;
            if (parentUL && parentUL.childElementCount === 1) {
                // 如果是最后一个子节点，隐藏父节点的展开箭头
                const parentLI = parentUL.previousElementSibling;
                if (parentLI && parentLI.tagName === "LI") {
                    const toggleElement = parentLI.querySelector(".b3-list-item__toggle");
                    if (toggleElement) {
                        toggleElement.classList.add("fn__hidden");
                    }
                    const arrowElement = parentLI.querySelector(".b3-list-item__arrow");
                    if (arrowElement) {
                        arrowElement.classList.remove("b3-list-item__arrow--open");
                    }
                }
                parentUL.remove();
            } else {
                targetElement.remove();
            }
        }
    }
};

export const deleteFile = (notebookId: string, pathString: string) => {
    if (window.siyuan.config.fileTree.removeDocWithoutConfirm) {
        fetchPost("/api/filetree/removeDoc", {
            notebook: notebookId,
            path: pathString
        }, () => {
            removeDocFromTree(pathString);
        });
        return;
    }
    fetchPost("/api/block/getDocInfo", {
        id: getDisplayName(pathString, true, true)
    }, (response) => {
        const fileName = escapeHtml(response.data.name);
        let tip = `${window.siyuan.languages.confirmDeleteTip.replace("${x}", fileName)}
<div class="fn__hr"></div>
<div class="ft__smaller ft__on-surface">${window.siyuan.languages.rollbackTip.replace("${x}", window.siyuan.config.editor.historyRetentionDays)}</div>`;
        if (response.data.subFileCount > 0) {
            tip = `${window.siyuan.languages.andSubFile.replace("${x}", fileName).replace("${y}", response.data.subFileCount)}
<div class="fn__hr"></div>
<div class="ft__smaller ft__on-surface">${window.siyuan.languages.rollbackTip.replace("${x}", window.siyuan.config.editor.historyRetentionDays)}</div>`;
        }
        confirmDialog(window.siyuan.languages.deleteOpConfirm, tip, () => {
            fetchPost("/api/filetree/removeDoc", {
                notebook: notebookId,
                path: pathString
            }, () => {
                removeDocFromTree(pathString);
            });
        }, undefined, true);
    });
};

export const deleteFiles = (liElements: Element[]) => {
    if (liElements.length === 1) {
        const itemTopULElement = hasTopClosestByTag(liElements[0], "UL");
        if (itemTopULElement) {
            const itemNotebookId = itemTopULElement.getAttribute("data-url");
            if (liElements[0].getAttribute("data-type") === "navigation-file") {
                deleteFile(itemNotebookId, liElements[0].getAttribute("data-path"));
            } else {
                confirmDialog(window.siyuan.languages.deleteOpConfirm,
                    `${window.siyuan.languages.confirmDeleteTip.replace("${x}", Lute.EscapeHTMLStr(getNotebookName(itemNotebookId)))}
<div class="fn__hr"></div>
<div class="ft__smaller ft__on-surface">${window.siyuan.languages.rollbackTip.replace("${x}", window.siyuan.config.editor.historyRetentionDays)}</div>`, () => {
                        fetchPost("/api/notebook/removeNotebook", {
                            notebook: itemNotebookId,
                            callback: Constants.CB_MOUNT_REMOVE
                        }, () => {
                            // 主动从文档树中移除笔记本，不依赖 WebSocket 推送
                            const fileModel = getDockByType("file")?.data?.file;
                            if (fileModel instanceof Files) {
                                const notebookElement = fileModel.element.querySelector(`ul[data-url="${itemNotebookId}"]`);
                                if (notebookElement) {
                                    notebookElement.remove();
                                }
                            }
                        });
                    }, undefined, true);
            }
        }
    } else {
        const paths: string[] = [];
        liElements.forEach(item => {
            const dataPath = item.getAttribute("data-path");
            if (dataPath !== "/") {
                paths.push(item.getAttribute("data-path"));
            }
        });
        if (paths.length === 0) {
            showMessage(window.siyuan.languages.notBatchRemove);
            return;
        }
        confirmDialog(window.siyuan.languages.deleteOpConfirm,
            `${window.siyuan.languages.confirmRemoveAll.replace("${count}", paths.length)}
<div class="fn__hr"></div>
<div class="ft__smaller ft__on-surface">${window.siyuan.languages.rollbackTip.replace("${x}", window.siyuan.config.editor.historyRetentionDays)}</div>`, () => {
                fetchPost("/api/filetree/removeDocs", {
                    paths
                }, () => {
                    // 批量删除后从文档树中移除所有文档
                    paths.forEach(path => removeDocFromTree(path));
                });
            }, undefined, true);
    }
};
