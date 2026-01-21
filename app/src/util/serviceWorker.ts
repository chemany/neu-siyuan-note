// https://github.com/siyuan-note/siyuan/pull/8012
export const registerServiceWorker = (
    scriptURL: string,
    options: RegistrationOptions = {
        scope: "/",
        type: "classic",
        updateViaCache: "all",
    },
) => {
    /// #if BROWSER
    if (window.webkit?.messageHandlers || window.JSAndroid || window.JSHarmony ||
        !("serviceWorker" in window.navigator)
        || !("caches" in window)
        || !("fetch" in window)
        || navigator.serviceWorker == null
    ) {
        return;
    }

    // 禁用 Service Worker：在通过 Nginx 代理访问时，Service Worker 的资源路径会出错
    // Service Worker 中的绝对路径（如 /favicon.ico）需要加上代理前缀（如 /notepads/favicon.ico）
    // 为了避免复杂的路径重写，暂时禁用 Service Worker
    console.log('[Service Worker] 已禁用 Service Worker 以避免代理路径问题');
    return;

    // REF https://developer.mozilla.org/en-US/docs/Web/API/ServiceWorkerRegistration
    // window.navigator.serviceWorker
    //     .register(scriptURL, options)
    //     .then(registration => {
    //         registration.update();
    //     }).catch(e => {
    //     console.debug(`Registration failed with ${e}`);
    // });
    /// #endif
};
