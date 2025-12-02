import {isLocalPath, pathPosix} from "../util/pathName";
/// #if !BROWSER
import {shell} from "electron";
/// #endif
import {getSearch} from "../util/functions";
import {openByMobile} from "../protyle/util/compatibility";
import {Constants} from "../constants";
import {showMessage} from "../dialog/message";
import {openAsset, openBy} from "./util";

// 需要双击才能预览的文档类型（Office 文档、Markdown 等）
const DBLCLICK_PREVIEW_EXTS = [
    ...Constants.SIYUAN_ASSETS_DOCUMENT,
    ".md", ".markdown", ".txt"
];

// 检查是否为需要双击预览的文档类型
export const isDocumentLink = (linkAddress: string): boolean => {
    if (!isLocalPath(linkAddress)) {
        return false;
    }
    const extName = pathPosix().extname(linkAddress).toLowerCase().split("?")[0];
    return DBLCLICK_PREVIEW_EXTS.includes(extName);
};

export const openLink = (protyle: IProtyle, aLink: string, event?: MouseEvent, ctrlIsPressed = false, isDblClick = false) => {
    let linkAddress = Lute.UnEscapeHTMLStr(aLink);
    let pdfParams;
    if (isLocalPath(linkAddress) && !linkAddress.startsWith("file://") && linkAddress.indexOf(".pdf") > -1) {
        const pdfAddress = linkAddress.split("/");
        if (pdfAddress.length === 3 && pdfAddress[0] === "assets" && pdfAddress[1].endsWith(".pdf") && /\d{14}-\w{7}/.test(pdfAddress[2])) {
            linkAddress = `assets/${pdfAddress[1]}`;
            pdfParams = pdfAddress[2];
        } else {
            pdfParams = parseInt(getSearch("page", linkAddress));
            linkAddress = linkAddress.split("?page")[0];
        }
    }
    /// #if MOBILE
    openByMobile(linkAddress);
    /// #else
    if (isLocalPath(linkAddress)) {
        const extName = pathPosix().extname(linkAddress).toLowerCase().split("?")[0];
        // 检查是否为支持内置预览的资源类型
        const isAssetExt = Constants.SIYUAN_ASSETS_EXTS.includes(extName);
        // 检查是否为需要双击才预览的文档类型
        const needDblClick = DBLCLICK_PREVIEW_EXTS.includes(extName);
        
        // 如果是需要双击预览的文档类型，单击时不做任何操作
        if (needDblClick && !isDblClick) {
            return;
        }
        
        if (isAssetExt || needDblClick) {
            // 所有支持的资源类型都使用内置预览
            if (event && event.altKey) {
                openAsset(protyle.app, linkAddress, pdfParams);
            } else if (event && event.shiftKey) {
                /// #if !BROWSER
                openBy(linkAddress, "app");
                /// #else
                // Web 模式下 Shift+点击也使用内置预览
                openAsset(protyle.app, linkAddress, pdfParams, "right");
                /// #endif
            } else if (ctrlIsPressed) {
                /// #if !BROWSER
                openBy(linkAddress, "folder");
                /// #else
                // Web 模式下 Ctrl+点击也使用内置预览
                openAsset(protyle.app, linkAddress, pdfParams, "right");
                /// #endif
            } else {
                // 默认在右侧打开预览
                openAsset(protyle.app, linkAddress, pdfParams, "right");
            }
        } else {
            /// #if !BROWSER
            if (ctrlIsPressed) {
                openBy(linkAddress, "folder");
            } else {
                openBy(linkAddress, "app");
            }
            /// #else
            openByMobile(linkAddress);
            /// #endif
        }
    } else if (linkAddress) {
        if (0 > linkAddress.indexOf(":")) {
            // 使用 : 判断，不使用 :// 判断 Open external application protocol invalid https://github.com/siyuan-note/siyuan/issues/10075
            // Support click to open hyperlinks like `www.foo.com` https://github.com/siyuan-note/siyuan/issues/9986
            linkAddress = `https://${linkAddress}`;
        }
        /// #if !BROWSER
        shell.openExternal(linkAddress).catch((e) => {
            showMessage(e);
        });
        /// #else
        openByMobile(linkAddress);
        /// #endif
    }
    /// #endif
};
