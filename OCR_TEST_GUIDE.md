# 本机请求 OCR 服务测试指南

本文档提供了一套用于在 Siyuan 服务器本机（156.x）测试和调试远程 OCR 服务（113.x）的详细指南。请使用这些命令来验证 OCR 服务的连通性、功能性和健壮性。

## 1. 基础环境检查

在开始测试之前，请确保基础网络连接正常。

### 1.1 健康检查 (Health Check)
验证服务是否在线且可访问。

```bash
curl -v -m 5 http://jason.cheman.top:8081/health
```

*   **预期结果**：HTTP 200 OK，返回 `{"status": "healthy", ...}`。
*   **常见错误**：
    *   `Connection refused`: 服务端未监听 0.0.0.0 或防火墙拦截。
    *   `Operation timed out`: 服务端处理阻塞或防火墙丢包。

## 2. 功能测试 (OCR Request)

使用真实图片数据测试 OCR 识别功能。

### 2.1 极简图片测试 (1x1 像素)
用于快速验证接口响应，能够复现 `axes don't match array` 错误。

```bash
curl -v -X POST http://jason.cheman.top:8081/ocr \
  -H "Content-Type: application/json" \
  -d '{"image": "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="}'
```

### 2.2 正常文字图片测试
使用包含真实文字的 Base64 图片进行测试。以下 Base64 对应一张包含 "Hello World" 的简单图片。

```bash
# 图片内容：白底黑字 "Hello World"
curl -v -X POST http://jason.cheman.top:8081/ocr \
  -H "Content-Type: application/json" \
  -d '{"image": "iVBORw0KGgoAAAANSUhEUgAAAGQAAAAyCAIAAACjTVvrAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAAAJcEhZcwAADsMAAA7DAcdvqGQAAACiSURBVGhD7dIxCwIxDAbg7f//0dXVDsVFBAcnD6U8cm1S09dI0tTz/Xwet+d53s/n7Xq9j8fjeT6f1/P5vN/v9/f7/f/+/wf8gQ0NEDY0QNjQAGFDA4QNDRA2NEDY0ABhQwOEDQ0QNjRA2NAAYUMDhA0NEDY0QNjQAGFDA4QNDRA2NEDY0ABhQwOEDQ0QNjRA2NAAYUMDhA0NEDY0QNjQAGFDA4QNDTAz8wBNqKA33c8FogAAAABJRU5ErkJggg=="}'
```

*   **预期结果**：HTTP 200 OK，返回包含 `results` 数组的 JSON，且识别出文字。

## 3. 错误排查与优化建议 (针对服务端)

如果遇到 `axes don't match array` 或其他 500 错误，建议在服务端各阶段添加日志。

### 3.1 关键调试点 (Python 代码示例)

请在您的服务端代码中检查以下环节：

```python
import base64
import numpy as np
import cv2
import logging

def process_request(json_data):
    try:
        # 1. 检查字段
        img_str = json_data.get('image')
        if not img_str:
            return {"status": "error", "message": "No image data found"}

        # 2. Base64 解码
        try:
            img_bytes = base64.b64decode(img_str)
        except Exception as e:
            return {"status": "error", "message": f"Base64 decode failed: {str(e)}"}

        if len(img_bytes) == 0:
            return {"status": "error", "message": "Empty image bytes"}

        # 3. 转换为 Numpy 数组
        np_arr = np.frombuffer(img_bytes, np.uint8)
        
        # 4. OpenCV 解码
        img = cv2.imdecode(np_arr, cv2.IMREAD_COLOR)
        
        # KEY CHECK: 检查解码是否成功
        if img is None:
             return {"status": "error", "message": "cv2.imdecode returned None. Invalid image data."}

        # 5. 打印图片维度 (调试用)
        logging.info(f"Image shape: {img.shape}")
        
        if img.shape[0] == 0 or img.shape[1] == 0:
            return {"status": "error", "message": "Image has 0 width or height"}

        # 6. 调用 PaddleOCR
        result = ocr.ocr(img, cls=True)
        return {"status": "success", "results": format_results(result)}

    except Exception as e:
        logging.error(f"OCR Error: {str(e)}", exc_info=True)
        return {"status": "error", "message": f"Server Error: {str(e)}"}
```

### 3.2 常见问题对照表

| 错误信息 | 可能原因 | 解决方案 |
| :--- | :--- | :--- |
| `Connection refused` | 服务绑定在 127.0.0.1 | 修改绑定地址为 `0.0.0.0` |
| `Operation timed out` | 防火墙拦截或服务卡死 | 检查 8081 端口入站规则，重启服务 |
| `axes don't match array` | cv2 解码失败或输入为空 | 增加 `if img is None` 检查，验证 Base64 字符串完整性 |
| `NoneType object is not subscriptable` | PaddleOCR 未检测到文字 | 增加对 `result` 为空的判断处理 |

## 4. Siyuan 客户端配置确认

确保 Siyuan 使用的配置文件 `/root/code/NeuraLink-Notes/config/ocr-config.json` 内容正确：

```json
{
  "baseUrl": "http://jason.cheman.top:8081",
  "enabled": true
}
```
