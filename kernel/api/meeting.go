package api

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/model"
)

// TranscribeAudio 这里的 handler 用于处理前端发送的转录请求
func TranscribeAudio(c *gin.Context) {
	// 获取上传的音频文件
	file, _, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "failed to get audio file"})
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "msg": "failed to read audio data"})
		return
	}

	// 调用服务层进行转录和摘要生成的处理
	result, err := model.Meeting.TranscribeAudio(audioData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "msg": err.Error()})
		return
	}

	// 返回 JSON 结果
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": result,
	})
}
