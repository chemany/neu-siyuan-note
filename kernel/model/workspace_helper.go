// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// GetRequestWorkspace 获取当前请求的workspace路径
// 在Web模式下,返回用户专属workspace
// 在普通模式下,返回全局workspace
func GetRequestWorkspace(c *gin.Context) string {
	// 尝试从context获取用户workspace
	if workspace, exists := c.Get("web_workspace"); exists {
		return workspace.(string)
	}

	// 返回全局workspace
	return util.WorkspaceDir
}

// GetRequestUserID 获取当前请求的用户ID
func GetRequestUserID(c *gin.Context) string {
	if userID, exists := c.Get("web_user_id"); exists {
		return userID.(string)
	}
	return ""
}

// GetRequestUsername 获取当前请求的用户名
func GetRequestUsername(c *gin.Context) string {
	if username, exists := c.Get("web_username"); exists {
		return username.(string)
	}
	return ""
}
