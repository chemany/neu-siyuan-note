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

package api

import (
	"net/http"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/conf"
	"github.com/siyuan-note/siyuan/kernel/model"
	"github.com/siyuan-note/siyuan/kernel/util"
)

func resetGraph(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	graph := conf.NewGlobalGraph()
	model.Conf.Graph.Global = graph
	model.Conf.Save()
	ret.Data = map[string]interface{}{
		"conf": graph,
	}
}

func resetLocalGraph(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	graph := conf.NewLocalGraph()
	model.Conf.Graph.Local = graph
	model.Conf.Save()
	ret.Data = map[string]interface{}{
		"conf": graph,
	}
}

// [关系图谱功能已禁用] getGraph 返回空数据
func getGraph(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	reqId := arg["reqId"]

	// 关系图谱功能已禁用，返回空数据
	ret.Data = map[string]interface{}{
		"nodes": []*model.GraphNode{},
		"links": []*model.GraphLink{},
		"conf":  model.Conf.Graph.Global,
		"box":   "",
		"reqId": reqId,
	}
}

// [关系图谱功能已禁用] getLocalGraph 返回空数据
func getLocalGraph(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	reqId := arg["reqId"]
	id := ""
	if arg["id"] != nil {
		id = arg["id"].(string)
	}

	// 关系图谱功能已禁用，返回空数据
	ret.Data = map[string]interface{}{
		"id":    id,
		"box":   "",
		"nodes": []*model.GraphNode{},
		"links": []*model.GraphLink{},
		"conf":  model.Conf.Graph.Local,
		"reqId": reqId,
	}
}
