import { Constants } from "../constants";
/// #if !BROWSER
import { ipcRenderer } from "electron";
/// #endif
import { processMessage } from "./processMessage";
import { kernelError } from "../dialog/processSystem";

// 获取cookie值的辅助函数
const getCookie = (name: string): string | null => {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) {
        return parts.pop()?.split(';').shift() || null;
    }
    return null;
};

export const fetchPost = (url: string, data?: any, cb?: (response: IWebSocketData) => void, headers?: IObject) => {
    const init: RequestInit = {
        method: "POST",
        headers: {
            'Content-Type': 'application/json',
            ...headers
        }
    };

    // 添加认证token（如果存在）
    const token = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
    if (token) {
        init.headers = {
            ...init.headers,
            'Authorization': `Bearer ${token}`
        };
    }
    if (data) {
        if (["/api/search/searchRefBlock", "/api/graph/getGraph", "/api/graph/getLocalGraph",
            "/api/block/getRecentUpdatedBlocks", "/api/search/fullTextSearchBlock"].includes(url)) {
            window.siyuan.reqIds[url] = new Date().getTime();
            if (data.type === "local" && url === "/api/graph/getLocalGraph") {
                // 当打开文档A的关系图、关系图、文档A后刷新，由于防止请求重复处理，文档A关系图无法渲染。
            } else {
                data.reqId = window.siyuan.reqIds[url];
            }
        }
        // 并发导出后端接受顺序不一致
        if (url === "/api/transactions") {
            data.reqId = new Date().getTime();
        }
        if (data instanceof FormData) {
            // FormData不需要Content-Type，让浏览器自动设置
            delete init.headers['Content-Type'];
            init.body = data;
        } else {
            init.body = JSON.stringify(data);
        }
    }

    // 如果传入headers，合并到现有headers中
    if (headers) {
        init.headers = {
            ...init.headers,
            ...headers
        };
    }
    fetch(url, init).then((response) => {
        switch (response.status) {
            case 403:
            case 404:
                return {
                    data: null,
                    msg: response.statusText,
                    code: -response.status,
                };
            default:
                if (401 == response.status) {
                    // 返回鉴权失败的话跳转到登录页，避免用户在当前页面操作 https://github.com/siyuan-note/siyuan/issues/15163
                    // 清除过期的token和用户信息
                    localStorage.removeItem('siyuan_token');
                    localStorage.removeItem('siyuan_user');
                    document.cookie = 'siyuan_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
                    
                    // 重定向到应用根路径，会自动跳转到登录页
                    setTimeout(() => {
                        window.location.href = window.location.origin + '/notepads/';
                    }, 1000);
                }

                if (response.headers.get("content-type")?.indexOf("application/json") > -1) {
                    return response.json();
                } else {
                    return response.text();
                }
        }
    }).then((response: IWebSocketData) => {
        if (typeof response === "string") {
            if (cb) {
                cb(response);
            }
            return;
        }
        if (["/api/search/searchRefBlock", "/api/graph/getGraph", "/api/graph/getLocalGraph",
            "/api/block/getRecentUpdatedBlocks", "/api/search/fullTextSearchBlock"].includes(url)) {
            if (response.data.reqId && window.siyuan.reqIds[url] && window.siyuan.reqIds[url] > response.data.reqId) {
                return;
            }
        }
        if (typeof response === "object" && typeof response.msg === "string" && typeof response.code === "number") {
            if (processMessage(response) && cb) {
                cb(response);
            }
        } else if (cb) {
            cb(response);
        }
    }).catch((e) => {
        console.warn("fetch post failed [" + e + "], url [" + url + "]");
        if (url === "/api/transactions" && (e.message === "Failed to fetch" || e.message === "Unexpected end of JSON input")) {
            kernelError();
            return;
        }
        /// #if !BROWSER
        if (url === "/api/system/exit" || url === "/api/system/setWorkspaceDir" || (
            ["/api/system/setUILayout"].includes(url) && data.errorExit // 内核中断，点关闭处理
        )) {
            ipcRenderer.send(Constants.SIYUAN_QUIT, location.port);
        }
        /// #endif
    });
};

export const fetchSyncPost = async (url: string, data?: any) => {
    const init: RequestInit = {
        method: "POST",
        headers: {
            'Content-Type': 'application/json'
        }
    };

    // 添加认证token（如果存在）
    const token = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
    if (token) {
        init.headers = {
            ...init.headers,
            'Authorization': `Bearer ${token}`
        };
    }
    if (data) {
        if (data instanceof FormData) {
            // FormData不需要Content-Type，让浏览器自动设置
            delete init.headers['Content-Type'];
            init.body = data;
        } else {
            init.body = JSON.stringify(data);
        }
    }
    const res = await fetch(url, init);
    const res2 = await res.json() as IWebSocketData;
    processMessage(res2);
    return res2;
};

export const fetchGet = (url: string, cb: (response: IWebSocketData | IObject | string) => void) => {
    const init: RequestInit = {
        method: "GET",
        headers: {}
    };
    const token = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
    if (token) {
        init.headers = {
            'Authorization': `Bearer ${token}`
        };
    }
    fetch(url, init).then((response) => {
        if (response.headers.get("content-type")?.indexOf("application/json") > -1) {
            return response.json();
        } else {
            return response.text();
        }
    }).then((response) => {
        cb(response);
    });
};
