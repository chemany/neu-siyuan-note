import * as md5 from "blueimp-md5";
import { hideMessage, showMessage } from "../dialog/message";
import { Constants } from "../constants";
import { fetchPost } from "../util/fetch";
import { repos } from "./repos";
import { confirmDialog } from "../dialog/confirmDialog";
import { hasClosestByClassName } from "../protyle/util/hasClosest";
import { getEventName, isInIOS } from "../protyle/util/compatibility";
import { processSync } from "../dialog/processSystem";
import { needSubscribe } from "../util/needSubscribe";
import { syncGuide } from "../sync/syncGuide";
import { hideElements } from "../protyle/ui/hideElements";
import { getCloudURL, getIndexURL } from "./util/about";
import { iOSPurchase } from "../util/iOSPurchase";

const getCookie = (name: string): string | null => {
    const cookies = document.cookie.split(';');
    for (let cookie of cookies) {
        const [cookieName, cookieValue] = cookie.trim().split('=');
        if (cookieName === name) {
            return cookieValue;
        }
    }
    return null;
};

const genSVGBG = () => {
    let html = "";
    const svgs: string[] = [];
    document.querySelectorAll("body > svg > defs > symbol").forEach((item) => {
        svgs.push(item.id);
    });
    Array.from({ length: 45 }, () => {
        const index = Math.floor(Math.random() * svgs.length);
        html += `<svg><use xlink:href="#${svgs[index]}"></use></svg>`;
        svgs.splice(index, 1);
    });
    return `<div class="fn__flex config-account__svg">${html}</div>`;
};

export const account = {
    element: undefined as Element,
    genHTML: (onlyPayHTML = false) => {
        const isIOS = isInIOS();
        let payHTML;
        if (isIOS) {
            // 已付费
            if (window.siyuan.user?.userSiYuanOneTimePayStatus === 1) {
                payHTML = `<button class="b3-button b3-button--big" data-action="iOSPay" data-type="subscribe">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account4}
</button>`;
            } else {
                payHTML = `<button class="b3-button b3-button--big" data-action="iOSPay" data-type="subscribe">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account10}
</button>
<div class="fn__hr"></div>
<button class="b3-button b3-button--success" data-action="iOSPay" data-type="function">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.onepay}
</button>`;
            }
        } else {
            payHTML = `<a class="b3-button b3-button--big" href="${getIndexURL("pricing.html")}" target="_blank">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages[window.siyuan.user?.userSiYuanOneTimePayStatus === 1 ? "account4" : "account1"]}
</a>`;
        }
        payHTML += `<div class="fn__hr--b"></div>
<span class="b3-chip b3-chip--primary b3-chip--hover${(window.siyuan.user && window.siyuan.user.userSiYuanSubscriptionStatus === 2) ? " fn__none" : ""}" id="trialSub">
    <svg class="ft__secondary"><use xlink:href="#iconVIP"></use></svg>
    ${window.siyuan.languages.freeSub}
</span>
<div class="fn__hr${(window.siyuan.user && window.siyuan.user.userSiYuanSubscriptionStatus === 2) ? " fn__none" : ""}"></div>
<a href="${getCloudURL("sponsor")}" target="_blank" class="${isIOS ? "fn__none " : ""}b3-chip b3-chip--pink b3-chip--hover">
    <svg version='1.1' xmlns='http://www.w3.org/2000/svg' width='32' height='32' viewBox='0 0 32 32'><path fill='#ffe43c' d='M6.4 0h19.2c4.268 0 6.4 2.132 6.4 6.4v19.2c0 4.268-2.132 6.4-6.4 6.4h-19.2c-4.268 0-6.4-2.132-6.4-6.4v-19.2c0-4.268 2.135-6.4 6.4-6.4z'></path> <path fill='#00f5d4' d='M25.6 0h-8.903c-7.762 1.894-14.043 7.579-16.697 15.113v10.487c0 3.533 2.867 6.4 6.4 6.4h19.2c3.533 0 6.4-2.867 6.4-6.4v-19.2c0-3.537-2.863-6.4-6.4-6.4z'></path> <path fill='#01beff' d='M25.6 0h-0.119c-12.739 2.754-20.833 15.316-18.079 28.054 0.293 1.35 0.702 2.667 1.224 3.946h16.974c3.533 0 6.4-2.867 6.4-6.4v-19.2c0-3.537-2.863-6.4-6.4-6.4z'></path> <path fill='#9a5ce5' d='M31.005 2.966c-0.457-0.722-1.060-1.353-1.784-1.849-8.342 3.865-13.683 12.223-13.679 21.416-0.003 3.256 0.67 6.481 1.978 9.463h8.081c0.602 0 1.185-0.084 1.736-0.238-2.1-3.189-3.401-7.624-3.401-12.526 0-7.337 2.921-13.628 7.070-16.266z'></path> <path fill='#f15bb5' d='M32 25.6v-19.2c0-1.234-0.354-2.419-0.998-3.43-4.149 2.638-7.067 8.928-7.067 16.266 0 4.902 1.301 9.334 3.401 12.526 2.693-0.757 4.664-3.231 4.664-6.162z'></path> <path fill='#fff' opacity='0.2' d='M26.972 22.415c-2.889 0.815-4.297 2.21-6.281 3.182 1.552 0.348 3.105 0.461 4.902 0.461 2.644 0 5.363-1.449 6.406-2.519v-1.085c-1.598-0.399-2.664-0.705-5.028-0.039zM4.773 21.612c-0.003 0-0.006-0.003-0.006-0.003-1.726-0.863-3.382-1.205-4.767-1.301v2.487c0.779-0.341 2.396-0.921 4.773-1.182zM17.158 26.599c1.472-0.158 2.57-0.531 3.533-1.002-1.063-0.238-2.126-0.583-3.269-1.079-2.767-1.205-5.63-3.092-10.491-3.034-0.779 0.010-1.495 0.058-2.158 0.132 4.503 2.248 7.882 5.463 12.384 4.983z'></path> <path fill='#fff' opacity='0.2' d='M20.691 25.594c-0.963 0.47-2.061 0.844-3.533 1.002-4.503 0.483-7.882-2.731-12.381-4.983-2.38 0.261-3.994 0.841-4.773 1.179v2.809c0 4.268 2.132 6.4 6.4 6.4h19.197c4.268 0 6.4-2.132 6.4-6.4v-2.065c-1.044 1.069-3.762 2.519-6.406 2.519-1.797 0-3.35-0.113-4.902-0.461z'></path> <path fill='#fff' opacity='0.5' d='M3.479 19.123c0 0.334 0.271 0.606 0.606 0.606s0.606-0.271 0.606-0.606v0c0-0.334-0.271-0.606-0.606-0.606s-0.606 0.271-0.606 0.606v0z'></path> <path fill='#fff' opacity='0.5' d='M29.027 14.266c0 0.334 0.271 0.606 0.606 0.606s0.606-0.271 0.606-0.606v0c0-0.334-0.271-0.606-0.606-0.606s-0.606 0.271-0.606 0.606v0z'></path> <path fill='#fff' d='M9.904 1.688c0 0.167 0.136 0.303 0.303 0.303s0.303-0.136 0.303-0.303v0c0-0.167-0.136-0.303-0.303-0.303s-0.303 0.136-0.303 0.303v0z'></path> <path fill='#fff' d='M2.673 10.468c0 0.167 0.136 0.303 0.303 0.303s0.303-0.136 0.303-0.303v0c0-0.167-0.136-0.303-0.303-0.303s-0.303 0.136-0.303 0.303v0z'></path> <path fill='#fff' opacity='0.6' d='M30.702 9.376c0 0.167 0.136 0.303 0.303 0.303s0.303-0.136 0.303-0.303v0c0-0.167-0.136-0.303-0.303-0.303s-0.303 0.136-0.303 0.303v0z'></path> <path fill='#fff' opacity='0.8' d='M29.236 20.881c0 0.276 0.224 0.499 0.499 0.499s0.499-0.224 0.499-0.499v0c0-0.276-0.224-0.499-0.499-0.499s-0.499 0.224-0.499 0.499v0z'></path> <path fill='#fff' opacity='0.8' d='M15.38 1.591c0.047 0.016 0.101 0.026 0.158 0.026 0.276 0 0.499-0.224 0.499-0.499 0-0.219-0.141-0.406-0.338-0.473l-0.004-0.001c-0.047-0.016-0.101-0.026-0.158-0.026-0.276 0-0.499 0.224-0.499 0.499 0 0.219 0.141 0.406 0.338 0.473l0.004 0.001z'></path> <path fill='#ffdeeb' d='M25.732 8.268c-2.393-2.371-6.249-2.371-8.642 0l-1.089 1.085-1.079-1.089c-2.38-2.39-6.249-2.393-8.639-0.013s-2.393 6.249-0.013 8.639l2.158 2.158 6.474 6.464c0.596 0.593 1.562 0.593 2.158 0l6.474-6.464 2.193-2.158c2.384-2.383 2.384-6.242 0.003-8.622z'></path> <path fill='#fff' d='M17.081 8.268l-1.079 1.085-1.079-1.089c-2.38-2.39-6.249-2.393-8.639-0.013s-2.393 6.249-0.013 8.639l2.158 2.158 2.548 2.487c4.097-1.044 7.627-3.646 9.837-7.254 1.424-2.271 2.284-4.848 2.503-7.518-2.193-0.715-4.606-0.132-6.236 1.504z'></path> </svg>
    ${window.siyuan.languages.sponsor}
</a>
<div class="fn__hr--b"></div>
<div class="fn__flex-1"></div>
<div>
${window.siyuan.languages.accountSupport1}
</div>
<div class="fn__hr--b"></div>
<div>
${window.siyuan.languages.accountSupport2}
</div>`;
        if (onlyPayHTML) {
            return `<div class="fn__flex-1 fn__hr--b"></div>
${genSVGBG()}
<div class="fn__flex-1 fn__hr--b"></div>    
${payHTML}
<div class="fn__flex-1 fn__hr--b"></div>
${genSVGBG()}
<div class="fn__flex-1 fn__hr--b"></div>`;
        }

        // 优先检查Web模式的JWT token
        const webToken = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
        if (webToken) {
            // 显示统一注册服务的账户信息
            let webUserData: any = null;
            try {
                const storedUser = localStorage.getItem('siyuan_user');
                if (storedUser) {
                    webUserData = JSON.parse(storedUser);
                }
            } catch (e) {
                console.error('Failed to parse web user data:', e);
            }

            if (webUserData) {
                // 生成用户头像的渐变色（基于用户名生成一致的颜色）
                const getAvatarGradient = (name: string) => {
                    const gradients = [
                        'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                        'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
                        'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
                        'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
                        'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
                        'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
                        'linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%)',
                        'linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%)'
                    ];
                    const hash = name ? name.charCodeAt(0) % gradients.length : 0;
                    return gradients[hash];
                };
                const avatarGradient = getAvatarGradient(webUserData.username || '');
                const avatarLetter = webUserData.username ? webUserData.username.charAt(0).toUpperCase() : 'U';
                
                return `<div class="fn__flex config-account" style="height: 100%; background: var(--b3-theme-background);">
    <!-- 左侧：用户资料卡片 -->
    <div class="config-account__center" style="flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: flex-start; padding: 40px 24px; overflow: auto;">
        <!-- 头像区域 -->
        <div style="position: relative; margin-bottom: 24px;">
            <div style="width: 120px; height: 120px; border-radius: 50%; background: ${avatarGradient}; display: flex; align-items: center; justify-content: center; box-shadow: 0 8px 32px rgba(102, 126, 234, 0.3); border: 4px solid var(--b3-theme-surface);">
                <span style="color: white; font-size: 48px; font-weight: 600; text-shadow: 0 2px 4px rgba(0,0,0,0.1);">${avatarLetter}</span>
            </div>
            <div style="position: absolute; bottom: 4px; right: 4px; width: 24px; height: 24px; background: #22c55e; border-radius: 50%; border: 3px solid var(--b3-theme-surface);"></div>
        </div>
        
        <!-- 用户名 -->
        <h2 style="margin: 0 0 8px 0; font-size: 24px; font-weight: 600; color: var(--b3-theme-on-background);">${webUserData.username || 'Unknown'}</h2>
        <span style="display: inline-flex; align-items: center; padding: 4px 12px; background: var(--b3-theme-primary-lightest); color: var(--b3-theme-primary); border-radius: 20px; font-size: 12px; font-weight: 500;">
            <svg style="width: 14px; height: 14px; margin-right: 4px;"><use xlink:href="#iconAccount"></use></svg>
            统一注册服务
        </span>
        
        <!-- 操作按钮 -->
        <div style="display: flex; gap: 12px; margin-top: 24px;">
            <button class="b3-button b3-button--outline" id="refreshWebProfile" style="display: flex; align-items: center; gap: 6px; padding: 8px 20px; border-radius: 8px;">
                <svg style="width: 16px; height: 16px;"><use xlink:href="#iconRefresh"></use></svg>
                刷新
            </button>
            <button class="b3-button b3-button--cancel" id="logoutWeb" style="display: flex; align-items: center; gap: 6px; padding: 8px 20px; border-radius: 8px;">
                <svg style="width: 16px; height: 16px;"><use xlink:href="#iconClose"></use></svg>
                ${window.siyuan.languages.logout}
            </button>
        </div>
        
        <!-- 信息卡片 -->
        <div style="width: 100%; max-width: 400px; margin-top: 32px; background: var(--b3-theme-surface); border-radius: 16px; padding: 24px; box-shadow: 0 2px 12px rgba(0,0,0,0.04);">
            <!-- 用户名 -->
            <div style="display: flex; align-items: center; padding: 16px 0; border-bottom: 1px solid var(--b3-border-color);">
                <div style="width: 40px; height: 40px; border-radius: 10px; background: var(--b3-theme-primary-lightest); display: flex; align-items: center; justify-content: center; margin-right: 16px;">
                    <svg style="width: 20px; height: 20px; color: var(--b3-theme-primary);"><use xlink:href="#iconAccount"></use></svg>
                </div>
                <div style="flex: 1;">
                    <div style="font-size: 12px; color: var(--b3-theme-on-surface-light); margin-bottom: 4px;">用户名</div>
                    <div style="font-size: 15px; color: var(--b3-theme-on-background); font-weight: 500;">${webUserData.username || 'N/A'}</div>
                </div>
            </div>
            
            <!-- 邮箱 -->
            <div style="display: flex; align-items: center; padding: 16px 0;">
                <div style="width: 40px; height: 40px; border-radius: 10px; background: rgba(34, 197, 94, 0.1); display: flex; align-items: center; justify-content: center; margin-right: 16px;">
                    <svg style="width: 20px; height: 20px; color: #22c55e;"><use xlink:href="#iconEmail"></use></svg>
                </div>
                <div style="flex: 1;">
                    <div style="font-size: 12px; color: var(--b3-theme-on-surface-light); margin-bottom: 4px;">邮箱地址</div>
                    <div style="font-size: 15px; color: var(--b3-theme-on-background); font-weight: 500;">${webUserData.email || 'N/A'}</div>
                </div>
            </div>
        </div>
    </div>
    
    <!-- 右侧：说明区域 -->
    <div class="config-account__center config-account__center--text" style="flex: 1; display: flex; flex-direction: column; justify-content: center; padding: 40px; background: var(--b3-theme-surface); overflow: auto;">
        <div style="max-width: 360px; margin: 0 auto;">
            ${genSVGBG()}
            <div style="margin: 32px 0;">
                <h3 style="margin: 0 0 16px 0; font-size: 18px; font-weight: 600; color: var(--b3-theme-on-background);">
                    <svg style="width: 20px; height: 20px; margin-right: 8px; vertical-align: middle;"><use xlink:href="#iconInfo"></use></svg>
                    关于统一注册服务
                </h3>
                <p style="margin: 0; font-size: 14px; line-height: 1.8; color: var(--b3-theme-on-surface);">
                    您的账户由<strong style="color: var(--b3-theme-primary);">统一注册服务</strong>管理，可以在多个应用之间共享登录状态。
                </p>
                <div style="margin-top: 20px; padding: 16px; background: var(--b3-theme-background); border-radius: 12px; border-left: 4px solid var(--b3-theme-primary);">
                    <div style="font-size: 13px; color: var(--b3-theme-on-surface-light);">
                        <div style="display: flex; align-items: center; margin-bottom: 8px;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            思源笔记
                        </div>
                        <div style="display: flex; align-items: center; margin-bottom: 8px;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            智能日历
                        </div>
                        <div style="display: flex; align-items: center;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            更多应用...
                        </div>
                    </div>
                </div>
            </div>
            ${genSVGBG()}
        </div>
    </div>
</div>`;
            }
        }

        // 如果没有webToken,检查思源云用户
        if (window.siyuan.user) {
            let userTitlesHTML = "";
            if (window.siyuan.user.userTitles.length > 0) {
                userTitlesHTML = '<div class="b3-chips" style="position: absolute">';
                window.siyuan.user.userTitles.forEach((item) => {
                    userTitlesHTML += `<div class="b3-chip b3-chip--middle b3-chip--primary">${item.icon} ${item.name}</div>`;
                });
                userTitlesHTML += "</div>";
            }
            let subscriptionHTML = "";
            let activeSubscriptionHTML = isIOS ? "" : `<div class="b3-form__icon fn__block">
   <svg class="ft__secondary b3-form__icon-icon"><use xlink:href="#iconVIP"></use></svg>
   <input class="b3-text-field fn__block b3-form__icon-input" style="padding-right: 44px;" placeholder="${window.siyuan.languages.activationCodePlaceholder}">
   <button id="activationCode" class="b3-button b3-button--text" style="position: absolute;right: 0;top: 0;">${window.siyuan.languages.confirm}</button>
</div>`;
            if (window.siyuan.user.userSiYuanProExpireTime === -1) {
                // 终身会员
                activeSubscriptionHTML = "";
                subscriptionHTML = `<div class="b3-chip b3-chip--secondary">${Constants.SIYUAN_IMAGE_VIP}${window.siyuan.languages.account12}</div>`;
            } else if (window.siyuan.user.userSiYuanProExpireTime > 0) {
                // 订阅中
                const renewHTML = `<div class="fn__hr--b"></div>
<div class="ft__on-surface ft__smaller">
    ${window.siyuan.languages.account6} 
    ${Math.max(0, Math.floor((window.siyuan.user.userSiYuanProExpireTime - new Date().getTime()) / 1000 / 60 / 60 / 24))} 
    ${window.siyuan.languages.day} 
    ${isIOS ? `<a href="javascript:void(0)" data-action="iOSPay" data-type="subscribe">${window.siyuan.languages.clickMeToRenew}</a>` : `<a href="${getCloudURL("subscribe/siyuan")}" target="_blank">${window.siyuan.languages.clickMeToRenew}</a>`}
</div>`;
                if (window.siyuan.user.userSiYuanOneTimePayStatus === 1) {
                    subscriptionHTML = `<div class="b3-chip b3-chip--success"><svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account7}</div>
<div class="fn__hr--b"></div>`;
                }
                if (window.siyuan.user.userSiYuanSubscriptionPlan === 2) {
                    // 订阅试用
                    subscriptionHTML += `<div class="b3-chip b3-chip--primary"><svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account3}</div>
${renewHTML}<div class="fn__hr--b"></div>`;
                } else {
                    // 年费
                    subscriptionHTML += `<div class="b3-chip b3-chip--primary"><svg class="ft__secondary"><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account8}</div>
${renewHTML}<div class="fn__hr--b"></div>`;
                }
                if (window.siyuan.user.userSiYuanOneTimePayStatus === 0) {
                    subscriptionHTML += isIOS ? `<button class="b3-button b3-button--success" data-action="iOSPay" data-type="function">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.onepay}
</button>` : `<a class="b3-button b3-button--success" href="${getIndexURL("pricing.html")}" target="_blank">
    <svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.onepay}
</a>`;
                }
            } else {
                if (window.siyuan.user.userSiYuanOneTimePayStatus === 1) {
                    subscriptionHTML = `<div class="b3-chip b3-chip--success"><svg><use xlink:href="#iconVIP"></use></svg>${window.siyuan.languages.account7}</div>
<div class="fn__hr--b"></div>${payHTML}`;
                } else {
                    subscriptionHTML = payHTML;
                }
            }
            return `<div class="fn__flex config-account">
<div class="config-account__center">
    <div class="config-account__bg">
        <div class="config-account__cover" style="background-image: url(${window.siyuan.user.userHomeBImgURL})"></div>
        <a href="${getCloudURL("settings/avatar")}" class="config-account__avatar" style="background-image: url(${window.siyuan.user.userAvatarURL})" target="_blank"></a>
        <h1 class="config-account__name">
            <a target="_blank" class="fn__a" href="${getCloudURL("member/" + window.siyuan.user.userName)}">${window.siyuan.user.userName}</a>
            <span class="ft__on-surface ft__smaller">${0 === window.siyuan.config.cloudRegion ? "ld246.com" : "liuyun.io"}</span>
        </h1>
        ${userTitlesHTML}
    </div>
    <div class="config-account__info">
        <div class="fn__flex">
            <a class="b3-button b3-button--text${isIOS ? " fn__none" : ""}" href="${getCloudURL("settings")}" target="_blank">${window.siyuan.languages.manage}</a>
            <span class="fn__space${isIOS ? " fn__none" : ""}"></span>
            <button class="b3-button b3-button--cancel" id="logout">
                ${window.siyuan.languages.logout}
            </button>
            <span class="fn__space"></span>
            <button class="b3-button b3-button--cancel${window.siyuan.config.system.container === "ios" ? "" : " fn__none"}" id="deactivateUser">
                ${window.siyuan.languages.deactivateUser}
            </button>
            <span class="fn__flex-1"></span>
            <button class="b3-button b3-button--cancel b3-tooltips b3-tooltips__n" id="refresh" aria-label="${window.siyuan.languages.refresh}">
                <svg style="margin-right: 0"><use xlink:href="#iconRefresh"></use></svg>
            </button>
        </div>
        <div class="fn__hr--b"></div>
        <div class="fn__flex">  
            <label>
                ${window.siyuan.languages.accountDisplayTitle}
                <input class="b3-switch fn__flex-center" id="displayTitle" type="checkbox"${window.siyuan.config.account.displayTitle ? " checked" : ""}/>
            </label>
            <div class="fn__flex-1"></div>
            <label>
                ${window.siyuan.languages.accountDisplayVIP}
                <input class="b3-switch fn__flex-center" id="displayVIP" type="checkbox"${window.siyuan.config.account.displayVIP ? " checked" : ""}/>
            </label>
        </div>
    </div>
</div>
<div class="config-account__center config-account__center--text">
    <div class="fn__flex-1 fn__hr--b"></div>
    ${subscriptionHTML}
    <div class="fn__flex-1 fn__hr--b"></div>
    ${activeSubscriptionHTML}
</div></div>`;
        }

        // 未登录状态 - 显示提示信息而不是登录表单
        return `<div class="fn__flex config-account" style="height: 100%; background: var(--b3-theme-background);">
    <!-- 左侧：登录提示 -->
    <div class="config-account__center" style="flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 40px 24px;">
        <div style="text-align: center; max-width: 360px;">
            <!-- 图标 -->
            <div style="width: 100px; height: 100px; margin: 0 auto 24px; border-radius: 50%; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; align-items: center; justify-content: center; box-shadow: 0 8px 32px rgba(102, 126, 234, 0.3);">
                <svg style="width: 48px; height: 48px; color: white;"><use xlink:href="#iconAccount"></use></svg>
            </div>
            
            <!-- 标题 -->
            <h2 style="margin: 0 0 12px 0; font-size: 24px; font-weight: 600; color: var(--b3-theme-on-background);">欢迎使用思源笔记</h2>
            <p style="margin: 0 0 32px 0; font-size: 15px; line-height: 1.6; color: var(--b3-theme-on-surface);">
                登录您的账户以同步数据和访问更多功能
            </p>
            
            <!-- 登录按钮 -->
            <button class="b3-button b3-button--big" onclick="window.location.href='/stage/login.html'" style="width: 100%; padding: 14px 32px; border-radius: 12px; font-size: 16px; font-weight: 500; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); border: none; box-shadow: 0 4px 16px rgba(102, 126, 234, 0.4);">
                <svg style="width: 18px; height: 18px; margin-right: 8px;"><use xlink:href="#iconAccount"></use></svg>
                立即登录
            </button>
            
            <p style="margin: 16px 0 0 0; font-size: 13px; color: var(--b3-theme-on-surface-light);">
                还没有账户？<a href="/stage/register.html" style="color: var(--b3-theme-primary); text-decoration: none; font-weight: 500;">立即注册</a>
            </p>
        </div>
    </div>
    
    <!-- 右侧：说明区域 -->
    <div class="config-account__center config-account__center--text" style="flex: 1; display: flex; flex-direction: column; justify-content: center; padding: 40px; background: var(--b3-theme-surface); overflow: auto;">
        <div style="max-width: 360px; margin: 0 auto;">
            ${genSVGBG()}
            <div style="margin: 32px 0;">
                <h3 style="margin: 0 0 16px 0; font-size: 18px; font-weight: 600; color: var(--b3-theme-on-background);">
                    <svg style="width: 20px; height: 20px; margin-right: 8px; vertical-align: middle;"><use xlink:href="#iconInfo"></use></svg>
                    关于统一注册服务
                </h3>
                <p style="margin: 0; font-size: 14px; line-height: 1.8; color: var(--b3-theme-on-surface);">
                    思源笔记使用<strong style="color: var(--b3-theme-primary);">统一注册服务</strong>进行账户管理，一个账户即可登录多个应用。
                </p>
                <div style="margin-top: 20px; padding: 16px; background: var(--b3-theme-background); border-radius: 12px; border-left: 4px solid var(--b3-theme-primary);">
                    <div style="font-size: 13px; color: var(--b3-theme-on-surface-light);">
                        <div style="display: flex; align-items: center; margin-bottom: 8px;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            思源笔记
                        </div>
                        <div style="display: flex; align-items: center; margin-bottom: 8px;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            智能日历
                        </div>
                        <div style="display: flex; align-items: center;">
                            <svg style="width: 16px; height: 16px; margin-right: 8px; color: var(--b3-theme-primary);"><use xlink:href="#iconMark"></use></svg>
                            更多应用...
                        </div>
                    </div>
                </div>
            </div>
            ${genSVGBG()}
        </div>
    </div>
</div>`;
    },
    bindEvent: (element: Element) => {
        // Web模式下的登出按钮处理
        const logoutWebButton = element.querySelector("#logoutWeb");
        if (logoutWebButton) {
            logoutWebButton.addEventListener("click", () => {
                if (!confirm('确定要退出登录吗？\n\n退出后您可以切换到其他账户登录。')) {
                    return;
                }

                const token = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
                if (token) {
                    // 调用登出API
                    fetchPost("/api/web/auth/logout", {}, () => {
                        // 清除本地token
                        localStorage.removeItem('siyuan_token');
                        localStorage.removeItem('siyuan_user');
                        document.cookie = 'siyuan_token=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';

                        showMessage('已退出登录', 3000);

                        // 跳转到登录页
                        setTimeout(() => {
                            window.location.href = '/stage/login.html';
                        }, 1000);
                    });
                }
            });
        }

        // Web模式下的刷新按钮处理
        const refreshWebButton = element.querySelector("#refreshWebProfile");
        if (refreshWebButton) {
            refreshWebButton.addEventListener("click", () => {
                const token = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
                if (token) {
                    fetchPost("/api/web/auth/profile", {}, (response) => {
                        if (response.code === 0 && response.data) {
                            localStorage.setItem('siyuan_user', JSON.stringify(response.data));
                            element.innerHTML = account.genHTML();
                            account.bindEvent(element);
                            showMessage('账户信息已刷新', 3000);
                        }
                    });
                }
            });
        }

        element.querySelectorAll('[data-action="iOSPay"]').forEach(item => {
            item.addEventListener("click", () => {
                iOSPurchase(item.getAttribute("data-type"));
            });
        });
        const trialSubElement = element.querySelector("#trialSub");
        if (trialSubElement) {
            trialSubElement.addEventListener("click", () => {
                fetchPost("/api/account/startFreeTrial", {}, () => {
                    const refreshBtn = element.querySelector("#refresh");
                    if (refreshBtn) {
                        refreshBtn.dispatchEvent(new Event("click"));
                    }
                });
            });
        }
        const agreeLoginElement = element.querySelector("#agreeLogin") as HTMLInputElement;
        const userNameElement = element.querySelector("#userName") as HTMLInputElement;
        
        // 如果是Web用户登录状态，不需要绑定思源云用户的事件
        const webToken = localStorage.getItem('siyuan_token') || getCookie('siyuan_token');
        if (webToken && element.querySelector("#logoutWeb")) {
            // Web用户的事件已经在上面绑定了，直接返回
            return;
        }
        
        if (!userNameElement) {
            const refreshElement = element.querySelector("#refresh");
            if (!refreshElement) {
                // 未登录状态，没有refresh按钮，直接返回
                return;
            }
            refreshElement.addEventListener("click", () => {
                const svgElement = refreshElement.firstElementChild;
                if (svgElement.classList.contains("fn__rotate")) {
                    return;
                }
                svgElement.classList.add("fn__rotate");
                fetchPost("/api/setting/getCloudUser", {
                    token: window.siyuan.user.userToken,
                }, response => {
                    window.siyuan.user = response.data;
                    element.innerHTML = account.genHTML();
                    account.bindEvent(element);
                    showMessage(window.siyuan.languages.refreshUser, 3000);
                    account.onSetaccount();
                    processSync();
                });
            });
            const logoutElement = element.querySelector("#logout");
            if (logoutElement) {
                logoutElement.addEventListener("click", () => {
                    fetchPost("/api/setting/logoutCloudUser", {}, () => {
                        fetchPost("/api/setting/getCloudUser", {}, response => {
                            window.siyuan.user = response.data;
                            element.innerHTML = account.genHTML();
                            account.bindEvent(element);
                            account.onSetaccount();
                            processSync();
                        });
                    });
                });
            }
            const deactivateElement = element.querySelector("#deactivateUser");
            if (deactivateElement) {
                deactivateElement.addEventListener(getEventName(), () => {
                    confirmDialog("⚠️ " + window.siyuan.languages.deactivateUser, window.siyuan.languages.deactivateUserTip, () => {
                        fetchPost("/api/account/deactivate", {}, () => {
                            window.siyuan.user = null;
                            element.innerHTML = account.genHTML();
                            account.bindEvent(element);
                            account.onSetaccount();
                            processSync();
                        });
                    });
                });
            }
            element.querySelectorAll("input[type='checkbox']").forEach(item => {
                item.addEventListener("change", () => {
                    fetchPost("/api/setting/setAccount", {
                        displayTitle: (element.querySelector("#displayTitle") as HTMLInputElement).checked,
                        displayVIP: (element.querySelector("#displayVIP") as HTMLInputElement).checked,
                    }, (response) => {
                        window.siyuan.config.account.displayTitle = response.data.displayTitle;
                        window.siyuan.config.account.displayVIP = response.data.displayVIP;
                        account.onSetaccount();
                    });
                });
            });
            const activationCodeElement = element.querySelector("#activationCode");
            activationCodeElement?.addEventListener("click", () => {
                const activationCodeInput = (activationCodeElement.previousElementSibling as HTMLInputElement);
                fetchPost("/api/account/checkActivationcode", { data: activationCodeInput.value }, (response) => {
                    if (0 !== response.code) {
                        activationCodeInput.value = "";
                    }
                    confirmDialog(window.siyuan.languages.activationCode, response.msg, () => {
                        if (response.code === 0) {
                            fetchPost("/api/account/useActivationcode", { data: (activationCodeElement.previousElementSibling as HTMLInputElement).value }, () => {
                                refreshElement.dispatchEvent(new CustomEvent("click"));
                            });
                        }
                    });
                });
            });
            return;
        }

        const userPasswordElement = element.querySelector("#userPassword") as HTMLInputElement;
        const captchaImgElement = element.querySelector("#captchaImg") as HTMLInputElement;
        const captchaElement = element.querySelector("#captcha") as HTMLInputElement;
        const twofactorAuthCodeElement = element.querySelector("#twofactorAuthCode") as HTMLInputElement;
        const loginBtnElement = element.querySelector("#login") as HTMLButtonElement;
        const login2BtnElement = element.querySelector("#login2") as HTMLButtonElement;
        agreeLoginElement.addEventListener("click", () => {
            if (agreeLoginElement.checked) {
                loginBtnElement.removeAttribute("disabled");
            } else {
                loginBtnElement.setAttribute("disabled", "disabled");
            }
        });
        userNameElement.focus();
        userNameElement.addEventListener("keydown", (event) => {
            if (event.isComposing) {
                event.preventDefault();
                return;
            }
            if (event.key === "Enter") {
                loginBtnElement.click();
                event.preventDefault();
            }
        });

        twofactorAuthCodeElement.addEventListener("keydown", (event) => {
            if (event.isComposing) {
                event.preventDefault();
                return;
            }
            if (event.key === "Enter") {
                login2BtnElement.click();
                event.preventDefault();
            }
        });

        captchaElement.addEventListener("keydown", (event) => {
            if (event.isComposing) {
                event.preventDefault();
                return;
            }
            if (event.key === "Enter") {
                loginBtnElement.click();
                event.preventDefault();
            }
        });
        userPasswordElement.addEventListener("keydown", (event) => {
            if (event.isComposing) {
                event.preventDefault();
                return;
            }
            if (event.key === "Enter") {
                loginBtnElement.click();
                event.preventDefault();
            }
        });
        let token: string;
        let needCaptcha: string;
        captchaImgElement.addEventListener("click", () => {
            captchaImgElement.setAttribute("src", getCloudURL("captcha") + `/login?needCaptcha=${needCaptcha}&t=${new Date().getTime()}`);
        });

        const cloudRegionElement = element.querySelector("#cloudRegion") as HTMLSelectElement;
        cloudRegionElement.addEventListener("change", () => {
            window.siyuan.config.cloudRegion = parseInt(cloudRegionElement.value);
            element.querySelector(".config-account__center--text").innerHTML = account.genHTML(true);
            element.querySelector("#form1").lastElementChild.innerHTML = `<a href="${getCloudURL("forget-pwd")}" class="b3-button b3-button--cancel" target="_blank">${window.siyuan.languages.forgetPassword}</a>
<span class="fn__space${window.siyuan.config.system.container === "ios" ? " fn__none" : ""}"></span>
<a href="${getCloudURL("register")}" class="b3-button b3-button--cancel${window.siyuan.config.system.container === "ios" ? " fn__none" : ""}" target="_blank">${window.siyuan.languages.register}</a>`;
        });
        loginBtnElement.addEventListener("click", () => {
            fetchPost("/api/account/login", {
                userName: userNameElement.value.replace(/(^\s*)|(\s*$)/g, ""),
                userPassword: md5(userPasswordElement.value),
                captcha: captchaElement.value.replace(/(^\s*)|(\s*$)/g, ""),
                cloudRegion: window.siyuan.config.cloudRegion,
            }, (data) => {
                let messageId;
                if (data.code === 1) {
                    messageId = showMessage(data.msg);
                    if (data.data.needCaptcha) {
                        // 验证码
                        needCaptcha = data.data.needCaptcha;
                        captchaElement.parentElement.classList.remove("fn__none");
                        captchaElement.previousElementSibling.setAttribute("src",
                            getCloudURL("captcha") + `/login?needCaptcha=${data.data.needCaptcha}`);
                        captchaElement.value = "";
                        return;
                    }
                    return;
                }
                if (data.code === 10) {
                    // 两步验证
                    element.querySelector("#form1").classList.add("fn__none");
                    element.querySelector("#form2").classList.remove("fn__none");
                    twofactorAuthCodeElement.focus();
                    token = data.data.token;
                    return;
                }
                hideMessage(messageId);
                fetchPost("/api/setting/getCloudUser", {
                    token: data.data.token,
                }, response => {
                    account._afterLogin(response, element);
                });
            });
        });

        login2BtnElement.addEventListener("click", () => {
            fetchPost("/api/setting/login2faCloudUser", {
                code: twofactorAuthCodeElement.value,
                token,
            }, response => {
                fetchPost("/api/setting/getCloudUser", {
                    token: response.data.token,
                }, userResponse => {
                    account._afterLogin(userResponse, element);
                });
            });
        });
    },
    _afterLogin(userResponse: IWebSocketData, element: Element) {
        window.siyuan.user = userResponse.data;
        processSync();
        element.innerHTML = account.genHTML();
        account.bindEvent(element);
        account.onSetaccount();
        if (element.getAttribute("data-action") === "go-repos") {
            if (needSubscribe("") && 0 === window.siyuan.config.sync.provider) {
                const dialogElement = hasClosestByClassName(element, "b3-dialog--open");
                if (dialogElement) {
                    dialogElement.querySelector('.b3-tab-bar [data-name="repos"]').dispatchEvent(new CustomEvent("click"));
                    element.removeAttribute("data-action");
                }
            } else {
                hideElements(["dialog"]);
                syncGuide();
            }
        }
    },
    onSetaccount() {
        if (repos.element) {
            repos.element.innerHTML = "";
        }
        // toolbarVIP 元素已移除，不再显示VIP状态图标
        // 所有功能已免费，无需显示订阅状态
    }
};
