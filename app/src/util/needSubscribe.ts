import { showMessage } from "../dialog/message";
import { getCloudURL } from "../config/util/about";

// 修改为总是返回false,不需要订阅即可使用所有功能
export const needSubscribe = (tip = window.siyuan.languages._kernel[29]) => {
    // 在Web模式下,所有功能免费开放
    return false;
};

// 修改为总是返回true,所有用户都视为已付费用户
export const isPaidUser = () => {
    return true;
};
