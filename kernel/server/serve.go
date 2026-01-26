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

package server

import (
	"bytes"
	"fmt"
	"html/template"
	"mime"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/88250/gulu"
	"github.com/emersion/go-webdav/caldav"
	"github.com/emersion/go-webdav/carddav"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
	"github.com/olahol/melody"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/api"
	"github.com/siyuan-note/siyuan/kernel/cmd"
	"github.com/siyuan-note/siyuan/kernel/model"
	"github.com/siyuan-note/siyuan/kernel/server/proxy"
	"github.com/siyuan-note/siyuan/kernel/util"
	"golang.org/x/net/webdav"
)

const (
	MethodMkCol     = "MKCOL"
	MethodCopy      = "COPY"
	MethodMove      = "MOVE"
	MethodLock      = "LOCK"
	MethodUnlock    = "UNLOCK"
	MethodPropFind  = "PROPFIND"
	MethodPropPatch = "PROPPATCH"
	MethodReport    = "REPORT"
)

var (
	sessionStore = cookie.NewStore([]byte("ATN51UlxVq1Gcvdf"))

	HttpMethods = []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
	WebDavMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,

		MethodMkCol,
		MethodCopy,
		MethodMove,
		MethodLock,
		MethodUnlock,
		MethodPropFind,
		MethodPropPatch,
	}
	CalDavMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,

		MethodMkCol,
		MethodCopy,
		MethodMove,
		// MethodLock,
		// MethodUnlock,
		MethodPropFind,
		MethodPropPatch,

		MethodReport,
	}
	CardDavMethods = []string{
		http.MethodOptions,
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,

		MethodMkCol,
		MethodCopy,
		MethodMove,
		// MethodLock,
		// MethodUnlock,
		MethodPropFind,
		MethodPropPatch,

		MethodReport,
	}
)

func Serve(fastMode bool) {
	gin.SetMode(gin.ReleaseMode)
	ginServer := gin.New()
	ginServer.UseH2C = true
	ginServer.MaxMultipartMemory = 1024 * 1024 * 32 // 插入较大的资源文件时内存占用较大 https://github.com/siyuan-note/siyuan/issues/5023
	ginServer.Use(
		model.ControlConcurrency, // 请求串行化 Concurrency control when requesting the kernel API https://github.com/siyuan-note/siyuan/issues/9939
		model.Timing,
		model.Recover,
		corsMiddleware(), // 后端服务支持 CORS 预检请求验证 https://github.com/siyuan-note/siyuan/pull/5593
		jwtMiddleware,    // 解析 JWT https://github.com/siyuan-note/siyuan/issues/11364
		gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{".pdf", ".mp3", ".wav", ".ogg", ".mov", ".weba", ".mkv", ".mp4", ".webm", ".flac"})),
	)

	sessionStore.Options(sessions.Options{
		Path:   "/",
		Secure: util.SSL,
		//MaxAge:   60 * 60 * 24 * 7, // 默认是 Session
		HttpOnly: true,
	})
	ginServer.Use(sessions.Sessions("siyuan", sessionStore))

	serveDebug(ginServer)
	serveAssets(ginServer)
	serveAppearance(ginServer)
	serveWebSocket(ginServer)
	serveWebDAV(ginServer)
	serveCalDAV(ginServer)
	serveCardDAV(ginServer)
	serveExport(ginServer)
	serveWidgets(ginServer)
	servePlugins(ginServer)
	serveEmojis(ginServer)
	serveTemplates(ginServer)
	servePublic(ginServer)
	serveSnippets(ginServer)
	serveRepoDiff(ginServer)
	serveCheckAuth(ginServer)
	serveFixedStaticFiles(ginServer)
	api.ServeAPI(ginServer)

	var host string
	// 强制监听所有网络接口，因为通过 Nginx 反向代理访问，安全性由 Nginx 和认证系统保证
	// 原逻辑: if model.Conf.System.NetworkServe || util.ContainerDocker == util.Container
	host = "0.0.0.0"

	ln, err := net.Listen("tcp", host+":"+util.ServerPort)
	if err != nil {
		if !fastMode {
			logging.LogErrorf("boot kernel failed: %s", err)
			os.Exit(logging.ExitCodeUnavailablePort)
		}

		// fast 模式下启动失败则直接返回
		return
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		if !fastMode {
			logging.LogErrorf("boot kernel failed: %s", err)
			os.Exit(logging.ExitCodeUnavailablePort)
		}
	}
	util.ServerPort = port

	util.ServerURL, err = url.Parse("http://127.0.0.1:" + port)
	if err != nil {
		logging.LogErrorf("parse server url failed: %s", err)
	}

	pid := fmt.Sprintf("%d", os.Getpid())
	if !fastMode {
		rewritePortJSON(pid, port)
	}
	logging.LogInfof("kernel [pid=%s] http server [%s] is booting", pid, host+":"+port)
	util.HttpServing = true

	go util.HookUILoaded()

	go func() {
		time.Sleep(1 * time.Second)
		go proxy.InitFixedPortService(host)
		go proxy.InitPublishService()
		// 反代服务器启动失败不影响核心服务器启动
	}()

	if err = http.Serve(ln, ginServer.Handler()); err != nil {
		if !fastMode {
			logging.LogErrorf("boot kernel failed: %s", err)
			os.Exit(logging.ExitCodeUnavailablePort)
		}
	}
}

func rewritePortJSON(pid, port string) {
	portJSON := filepath.Join(util.HomeDir, ".config", "siyuan", "port.json")
	pidPorts := map[string]string{}
	var data []byte
	var err error

	if gulu.File.IsExist(portJSON) {
		data, err = os.ReadFile(portJSON)
		if err != nil {
			logging.LogWarnf("read port.json failed: %s", err)
		} else {
			if err = gulu.JSON.UnmarshalJSON(data, &pidPorts); err != nil {
				logging.LogWarnf("unmarshal port.json failed: %s", err)
			}
		}
	}

	pidPorts[pid] = port
	if data, err = gulu.JSON.MarshalIndentJSON(pidPorts, "", "  "); err != nil {
		logging.LogWarnf("marshal port.json failed: %s", err)
	} else {
		if err = os.WriteFile(portJSON, data, 0644); err != nil {
			logging.LogWarnf("write port.json failed: %s", err)
		}
	}
}

func serveExport(ginServer *gin.Engine) {
	// Potential data export disclosure security vulnerability https://github.com/siyuan-note/siyuan/issues/12213
	exportGroup := ginServer.Group("/export/", model.CheckAuth)
	exportGroup.Static("/", filepath.Join(util.TempDir, "export"))
}

func serveWidgets(ginServer *gin.Engine) {
	ginServer.Static("/widgets/", filepath.Join(util.DataDir, "widgets"))
}

func servePlugins(ginServer *gin.Engine) {
	ginServer.Static("/plugins/", filepath.Join(util.DataDir, "plugins"))
}

func serveEmojis(ginServer *gin.Engine) {
	ginServer.Static("/emojis/", filepath.Join(util.DataDir, "emojis"))
}

func serveTemplates(ginServer *gin.Engine) {
	ginServer.Static("/templates/", filepath.Join(util.DataDir, "templates"))
}

func servePublic(ginServer *gin.Engine) {
	// Support directly access `data/public/*` contents via URL link https://github.com/siyuan-note/siyuan/issues/8593
	ginServer.Static("/public/", filepath.Join(util.DataDir, "public"))
}

func serveSnippets(ginServer *gin.Engine) {
	ginServer.Handle("GET", "/snippets/*filepath", func(c *gin.Context) {
		filePath := strings.TrimPrefix(c.Request.URL.Path, "/snippets/")
		ext := filepath.Ext(filePath)
		name := strings.TrimSuffix(filePath, ext)
		confSnippets, err := model.LoadSnippets()
		if err != nil {
			logging.LogErrorf("load snippets failed: %s", err)
			c.Status(http.StatusNotFound)
			return
		}

		for _, s := range confSnippets {
			if s.Name == name && ("" != ext && s.Type == ext[1:]) {
				c.Header("Content-Type", mime.TypeByExtension(ext))
				c.String(http.StatusOK, s.Content)
				return
			}
		}

		// 没有在配置文件中命中时在文件系统上查找
		filePath = filepath.Join(util.SnippetsPath, filePath)
		c.File(filePath)
	})
}

func serveAppearance(ginServer *gin.Engine) {
	siyuan := ginServer.Group("", model.CheckWebAuth) // 使用Web认证中间件

	siyuan.Handle("GET", "/", func(c *gin.Context) {
		userAgentHeader := c.GetHeader("User-Agent")
		logging.LogInfof("serving [/] for user-agent [%s]", userAgentHeader)

		// Carry query parameters when redirecting
		location := url.URL{}
		queryParams := c.Request.URL.Query()
		queryParams.Set("r", gulu.Rand.String(7))
		location.RawQuery = queryParams.Encode()

		if strings.Contains(userAgentHeader, "Electron") {
			location.Path = "/stage/build/app/"
		} else if strings.Contains(userAgentHeader, "Pad") ||
			(strings.ContainsAny(userAgentHeader, "Android") && !strings.Contains(userAgentHeader, "Mobile")) {
			// Improve detecting Pad device, treat it as desktop device https://github.com/siyuan-note/siyuan/issues/8435 https://github.com/siyuan-note/siyuan/issues/8497
			location.Path = "/stage/build/desktop/"
		} else {
			if idx := strings.Index(userAgentHeader, "Mozilla/"); 0 < idx {
				userAgentHeader = userAgentHeader[idx:]
			}
			ua := useragent.New(userAgentHeader)
			if ua.Mobile() {
				location.Path = "/stage/build/mobile/"
			} else {
				location.Path = "/stage/build/desktop/"
			}
		}

		c.Redirect(302, location.String())
	})

	siyuan.GET("/appearance/*filepath", func(c *gin.Context) {
		// 获取用户的 WorkspaceContext
		ctx := model.GetWorkspaceContext(c)
		
		// 根据请求路径确定使用哪个 appearance 目录
		// 主题和图标从全局目录读取（所有用户共享）
		// 语言文件从用户目录读取（用户特定，需要认证）
		var appearancePath string
		requestPath := c.Request.URL.Path
		
		if strings.Contains(requestPath, "/langs/") {
			// 语言文件从用户的 conf/appearance 目录读取
			// 注意：此路径需要认证，所以 ctx 一定是用户特定的
			appearancePath = filepath.Join(ctx.GetConfDir(), "appearance")
		} else if strings.Contains(requestPath, "/themes/") || strings.Contains(requestPath, "/icons/") {
			// 主题和图标从全局目录读取
			if "dev" == util.Mode {
				appearancePath = filepath.Join(util.WorkingDir, "appearance")
			} else {
				appearancePath = util.AppearancePath
			}
		} else {
			// 其他文件（如 boot、fonts、emojis 等）从全局目录读取
			if "dev" == util.Mode {
				appearancePath = filepath.Join(util.WorkingDir, "appearance")
			} else {
				appearancePath = util.AppearancePath
			}
		}
		
		filePath := filepath.Join(appearancePath, strings.TrimPrefix(c.Request.URL.Path, "/appearance/"))
		if strings.HasSuffix(c.Request.URL.Path, "/theme.js") {
			if !gulu.File.IsExist(filePath) {
				// 主题 js 不存在时生成空内容返回
				c.Data(200, "application/x-javascript", nil)
				return
			}
		} else if strings.Contains(c.Request.URL.Path, "/langs/") && strings.HasSuffix(c.Request.URL.Path, ".json") {
			lang := path.Base(c.Request.URL.Path)
			lang = strings.TrimSuffix(lang, ".json")
			if "zh_CN" != lang && "en_US" != lang {
				// 多语言配置缺失项使用对应英文配置项补齐 https://github.com/siyuan-note/siyuan/issues/5322

				enUSFilePath := filepath.Join(appearancePath, "langs", "en_US.json")
				enUSData, err := os.ReadFile(enUSFilePath)
				if err != nil {
					logging.LogErrorf("read en_US.json [%s] failed: %s", enUSFilePath, err)
					util.ReportFileSysFatalError(err)
					return
				}
				enUSMap := map[string]interface{}{}
				if err = gulu.JSON.UnmarshalJSON(enUSData, &enUSMap); err != nil {
					logging.LogErrorf("unmarshal en_US.json [%s] failed: %s", enUSFilePath, err)
					util.ReportFileSysFatalError(err)
					return
				}

				for {
					data, err := os.ReadFile(filePath)
					if err != nil {
						c.JSON(200, enUSMap)
						return
					}

					langMap := map[string]interface{}{}
					if err = gulu.JSON.UnmarshalJSON(data, &langMap); err != nil {
						logging.LogErrorf("unmarshal json [%s] failed: %s", filePath, err)
						c.JSON(200, enUSMap)
						return
					}

					for enUSDataKey, enUSDataValue := range enUSMap {
						if _, ok := langMap[enUSDataKey]; !ok {
							langMap[enUSDataKey] = enUSDataValue
						}
					}
					c.JSON(200, langMap)
					return
				}
			}
		}

		c.File(filePath)
	})

	siyuan.Static("/stage", filepath.Join(util.WorkingDir, "stage"))
}

func serveCheckAuth(ginServer *gin.Engine) {
	ginServer.GET("/check-auth", serveAuthPage)
}

func serveAuthPage(c *gin.Context) {
	data, err := os.ReadFile(filepath.Join(util.WorkingDir, "stage/auth.html"))
	if err != nil {
		logging.LogErrorf("load auth page failed: %s", err)
		c.Status(500)
		return
	}

	tpl, err := template.New("auth").Parse(string(data))
	if err != nil {
		logging.LogErrorf("parse auth page failed: %s", err)
		c.Status(500)
		return
	}

	keymapHideWindow := "⌥M"
	if nil != (*model.Conf.Keymap)["general"] {
		switch (*model.Conf.Keymap)["general"].(type) {
		case map[string]interface{}:
			keymapGeneral := (*model.Conf.Keymap)["general"].(map[string]interface{})
			if nil != keymapGeneral["toggleWin"] {
				switch keymapGeneral["toggleWin"].(type) {
				case map[string]interface{}:
					toggleWin := keymapGeneral["toggleWin"].(map[string]interface{})
					if nil != toggleWin["custom"] {
						keymapHideWindow = toggleWin["custom"].(string)
					}
				}
			}
		}
		if "" == keymapHideWindow {
			keymapHideWindow = "⌥M"
		}
	}
	model := map[string]interface{}{
		"l0":                     model.Conf.Language(173),
		"l1":                     model.Conf.Language(174),
		"l2":                     template.HTML(model.Conf.Language(172)),
		"l3":                     model.Conf.Language(175),
		"l4":                     model.Conf.Language(176),
		"l5":                     model.Conf.Language(177),
		"l6":                     model.Conf.Language(178),
		"l7":                     template.HTML(model.Conf.Language(184)),
		"l8":                     model.Conf.Language(95),
		"l9":                     model.Conf.Language(83),
		"l10":                    model.Conf.Language(257),
		"appearanceMode":         model.Conf.Appearance.Mode,
		"appearanceModeOS":       model.Conf.Appearance.ModeOS,
		"workspace":              util.WorkspaceName,
		"workspacePath":          util.WorkspaceDir,
		"keymapGeneralToggleWin": keymapHideWindow,
		"trayMenuLangs":          util.TrayMenuLangs[util.Lang],
		"workspaceDir":           util.WorkspaceDir,
	}
	buf := &bytes.Buffer{}
	if err = tpl.Execute(buf, model); err != nil {
		logging.LogErrorf("execute auth page failed: %s", err)
		c.Status(500)
		return
	}
	data = buf.Bytes()
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

func serveAssets(ginServer *gin.Engine) {
	// 注册 /upload 端点作为 /api/asset/upload 的别名
	// 在Web模式下使用 CheckWebAuth，在传统模式下使用 CheckAuth
	if os.Getenv("SIYUAN_WEB_MODE") == "true" {
		ginServer.POST("/upload", model.CheckWebAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	} else {
		ginServer.POST("/upload", model.CheckAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	}

	// 公开的 assets 访问端点，用于 Office Online Viewer 等外部服务访问
	// 仅支持 Office 文档类型，避免敏感文件泄露
	publicAssetsHandler := func(context *gin.Context) {
		requestPath := context.Param("path")
		if "/" == requestPath || "" == requestPath {
			context.Status(http.StatusForbidden)
			return
		}

		// URL 解码文件名（处理中文等特殊字符）
		decodedPath, err := url.PathUnescape(requestPath)
		if err != nil {
			decodedPath = requestPath
		}

		// 仅允许访问 Office 文档类型
		ext := strings.ToLower(filepath.Ext(decodedPath))
		allowedExts := map[string]bool{
			".doc": true, ".docx": true,
			".xls": true, ".xlsx": true,
			".ppt": true, ".pptx": true,
			".odt": true, ".ods": true, ".odp": true,
		}
		if !allowedExts[ext] {
			logging.LogWarnf("public-assets: forbidden extension [%s] for path [%s]", ext, decodedPath)
			context.Status(http.StatusForbidden)
			return
		}

		// 在多用户模式下，搜索所有用户的 assets 目录
		userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
		if userDataRoot == "" {
			userDataRoot = "/root/code/MindOcean/user-data/notes"
		}

		fileName := strings.TrimPrefix(decodedPath, "/")
		var foundPath string

		logging.LogInfof("public-assets: searching for file [%s] in [%s]", fileName, userDataRoot)

		// 遍历用户数据目录查找文件
		filepath.Walk(userDataRoot, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			// 检查文件名是否匹配（在 assets 目录下）
			baseName := filepath.Base(p)
			if baseName == fileName && strings.Contains(p, "/assets/") {
				foundPath = p
				logging.LogInfof("public-assets: found file at [%s]", p)
				return filepath.SkipAll
			}
			return nil
		})

		if foundPath == "" {
			logging.LogWarnf("public-assets: file not found [%s]", fileName)
			context.Status(http.StatusNotFound)
			return
		}

		// 设置正确的 Content-Type
		contentType := mime.TypeByExtension(ext)
		if contentType != "" {
			context.Header("Content-Type", contentType)
		}
		// 允许跨域访问（Office Online Viewer 需要）
		context.Header("Access-Control-Allow-Origin", "*")

		http.ServeFile(context.Writer, context.Request, foundPath)
	}
	ginServer.GET("/public-assets/*path", publicAssetsHandler)
	ginServer.HEAD("/public-assets/*path", publicAssetsHandler)

	ginServer.GET("/assets/*path", model.CheckWebAuth, func(context *gin.Context) {
		requestPath := context.Param("path")
		if "/" == requestPath || "" == requestPath {
			// 禁止访问根目录 Disable HTTP access to the /assets/ path https://github.com/siyuan-note/siyuan/issues/15257
			context.Status(http.StatusForbidden)
			return
		}

		relativePath := path.Join("assets", requestPath)
		p, err := model.GetAssetAbsPath(relativePath)
		if err != nil {
			// 如果标准路径找不到，尝试从当前用户目录下搜索
			// 支持 /assets/xxx.png 格式，自动映射到 /assets/{username}/xxx.png
			if !strings.Contains(requestPath, "/") {
				// 请求格式: /assets/xxx.png（没有用户名前缀）
				username, exists := context.Get("web_username")
				if exists && username != "" {
					userAssetsPath := filepath.Join(util.DataDir, username.(string), "assets", requestPath)
					if gulu.File.IsExist(userAssetsPath) {
						p = userAssetsPath
						err = nil
					}
				}
			}

			if err != nil {
				if strings.Contains(strings.TrimPrefix(requestPath, "/"), "/") {
					// 再使用编码过的路径解析一次 https://github.com/siyuan-note/siyuan/issues/11823
					dest := url.PathEscape(strings.TrimPrefix(requestPath, "/"))
					dest = strings.ReplaceAll(dest, ":", "%3A")
					relativePath = path.Join("assets", dest)
					p, err = model.GetAssetAbsPath(relativePath)
				}

				if err != nil {
					context.Status(http.StatusNotFound)
					return
				}
			}
		}

		if serveThumbnail(context, p, requestPath) {
			// 如果请求缩略图服务成功则返回
			return
		}

		// 返回原始文件
		http.ServeFile(context.Writer, context.Request, p)
		return
	})

	ginServer.GET("/history/*path", model.CheckAuth, model.CheckAdminRole, func(context *gin.Context) {
		p := filepath.Join(util.HistoryDir, context.Param("path"))
		http.ServeFile(context.Writer, context.Request, p)
		return
	})
}

func serveThumbnail(context *gin.Context, assetAbsPath, requestPath string) bool {
	if style := context.Query("style"); style == "thumb" && model.NeedGenerateAssetsThumbnail(assetAbsPath) { // 请求缩略图
		thumbnailPath := filepath.Join(util.TempDir, "thumbnails", "assets", requestPath)
		if !gulu.File.IsExist(thumbnailPath) {
			// 如果缩略图不存在，则生成缩略图
			err := model.GenerateAssetsThumbnail(assetAbsPath, thumbnailPath)
			if err != nil {
				logging.LogErrorf("generate thumbnail failed: %s", err)
				return false
			}
		}

		http.ServeFile(context.Writer, context.Request, thumbnailPath)
		return true
	}
	return false
}

func serveRepoDiff(ginServer *gin.Engine) {
	ginServer.GET("/repo/diff/*path", model.CheckAuth, model.CheckAdminRole, func(context *gin.Context) {
		requestPath := context.Param("path")
		p := filepath.Join(util.TempDir, "repo", "diff", requestPath)
		http.ServeFile(context.Writer, context.Request, p)
		return
	})
}

func serveDebug(ginServer *gin.Engine) {
	if "prod" == util.Mode {
		// The production environment will no longer register `/debug/pprof/` https://github.com/siyuan-note/siyuan/issues/10152
		return
	}

	ginServer.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/allocs", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/block", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/goroutine", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/heap", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/mutex", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/threadcreate", gin.WrapF(pprof.Index))
	ginServer.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	ginServer.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	ginServer.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	ginServer.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))
}

func serveWebSocket(ginServer *gin.Engine) {
	util.WebSocketServer.Config.MaxMessageSize = 1024 * 1024 * 8

	ginServer.GET("/ws", func(c *gin.Context) {
		if err := util.WebSocketServer.HandleRequest(c.Writer, c.Request); err != nil {
			logging.LogErrorf("handle command failed: %s", err)
		}
	})

	util.WebSocketServer.HandlePong(func(session *melody.Session) {
		//logging.LogInfof("pong")
	})

	util.WebSocketServer.HandleConnect(func(s *melody.Session) {
		//logging.LogInfof("ws check auth for [%s]", s.Request.RequestURI)
		authOk := false
		var workspaceCtx *model.WorkspaceContext // 用于存储用户的 WorkspaceContext
		var userID string
		var username string

		// Web Mode下优先使用JWT认证
		if os.Getenv("SIYUAN_WEB_MODE") == "true" {
			if token := model.ParseXAuthToken(s.Request); token != nil {
				// 先检查 token 是否有效
				if !token.Valid {
					logging.LogWarnf("[WebSocket] Invalid token")
				} else {
					// Token 有效，检查是否是灵枢笔记 token（有 role 字段且为数字）
					claims := model.GetTokenClaims(token)
					if role, ok := claims[model.ClaimsKeyRole]; ok {
						// 灵枢笔记 token，检查 role
						if roleFloat, ok := role.(float64); ok {
							authOk = model.IsValidRole(model.Role(roleFloat), []model.Role{
								model.RoleAdministrator,
								model.RoleEditor,
								model.RoleReader,
							})
							logging.LogInfof("[WebSocket] SiYuan token validated, role: %v, authOk: %v", roleFloat, authOk)
						}
					} else {
						// 没有 role 字段，可能是统一认证 token，直接允许
						authOk = true
						logging.LogInfof("[WebSocket] Unified auth token detected, allowing connection")
					}
				}
				
				// ✅ 从 token 中提取用户信息并创建 WorkspaceContext
				if authOk {
					claims := model.GetTokenClaims(token)
					
					// 尝试从 claims 中提取 workspace（灵枢笔记 token）
					if workspace, ok := claims["workspace"].(string); ok && workspace != "" {
						if uid, ok := claims["user_id"].(string); ok {
							userID = uid
						}
						if uname, ok := claims["username"].(string); ok {
							username = uname
						}
						workspaceCtx = model.NewWorkspaceContextWithUser(workspace, userID, username)
						logging.LogInfof("[WebSocket] Connected with SiYuan token - user: %s, workspace: %s", username, workspace)
					} else {
						// 如果没有 workspace，说明是统一认证 token，需要验证并获取用户信息
						unifiedService := model.GetUnifiedAuthService()
						if unifiedService != nil {
							// 从 token 字符串重新解析（因为 ParseXAuthToken 已经验证过了）
							tokenStr := ""
							authHeader := s.Request.Header.Get("Authorization")
							if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
								tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
							}
							if tokenStr == "" {
								// 从 URL 参数获取
								tokenStr = s.Request.URL.Query().Get("token")
							}
							
							if tokenStr != "" {
								unifiedUser, err := unifiedService.VerifyToken(tokenStr)
								if err == nil && unifiedUser != nil {
									// 确保本地用户存在
									localUser, err := unifiedService.EnsureLocalUser(unifiedUser)
									if err == nil && localUser != nil {
										userID = localUser.ID
										username = localUser.Username
										workspaceCtx = model.NewWorkspaceContextWithUser(localUser.Workspace, userID, username)
										logging.LogInfof("[WebSocket] Connected with unified token - user: %s, workspace: %s", username, localUser.Workspace)
									} else {
										logging.LogErrorf("[WebSocket] Failed to ensure local user: %s", err)
									}
								} else {
									logging.LogErrorf("[WebSocket] Failed to verify unified token: %s", err)
								}
							}
						}
					}
				}
			}
		} else {
			// 传统模式下的认证逻辑
			if "" != model.Conf.AccessAuthCode {
				session, err := sessionStore.Get(s.Request, "siyuan")
				if err != nil {
					authOk = false
					logging.LogErrorf("get cookie failed: %s", err)
				} else {
					val := session.Values["data"]
					if nil == val {
						authOk = false
					} else {
						sess := &util.SessionData{}
						err = gulu.JSON.UnmarshalJSON([]byte(val.(string)), sess)
						if err != nil {
							authOk = false
							logging.LogErrorf("unmarshal cookie failed: %s", err)
						} else {
							workspaceSess := util.GetWorkspaceSession(sess)
							authOk = workspaceSess.AccessAuthCode == model.Conf.AccessAuthCode
						}
					}
				}
			}

			// REF: https://github.com/siyuan-note/siyuan/issues/11364
			if !authOk {
				if token := model.ParseXAuthToken(s.Request); token != nil {
					authOk = token.Valid && model.IsValidRole(model.GetClaimRole(model.GetTokenClaims(token)), []model.Role{
						model.RoleAdministrator,
						model.RoleEditor,
						model.RoleReader,
					})
					
					// ✅ 从 token 中提取用户信息
					if authOk {
						claims := model.GetTokenClaims(token)
						if workspace, ok := claims["workspace"].(string); ok && workspace != "" {
							if uid, ok := claims["user_id"].(string); ok {
								userID = uid
							}
							if uname, ok := claims["username"].(string); ok {
								username = uname
							}
							workspaceCtx = model.NewWorkspaceContextWithUser(workspace, userID, username)
							logging.LogInfof("[WebSocket] Connected with token - user: %s, workspace: %s", username, workspace)
						}
					}
				}
			}
		}

		if !authOk {
			// 用于授权页保持连接，避免非常驻内存内核自动退出 https://github.com/siyuan-note/insider/issues/1099
			authOk = strings.Contains(s.Request.RequestURI, "/ws?app=siyuan&id=auth")
		}

		if !authOk {
			s.CloseWithMsg([]byte("  unauthenticated"))
			logging.LogWarnf("closed an unauthenticated session [%s]", util.GetRemoteAddr(s.Request))
			return
		}

		// ✅ 将 WorkspaceContext 存储到 session 中
		if workspaceCtx != nil {
			s.Set("workspaceContext", workspaceCtx)
			s.Set("web_user_id", userID)
			s.Set("web_username", username)
			s.Set("web_workspace", workspaceCtx.DataDir)
			
			// 设置为当前用户的 Context
			model.SetCurrentUserContext(username, workspaceCtx)
			logging.LogInfof("[WebSocket] Set current user context for user: %s", username)
			logging.LogInfof("[WebSocket] Stored WorkspaceContext - user: %s, workspace: %s", username, workspaceCtx.DataDir)
		} else {
			logging.LogWarnf("[WebSocket] No WorkspaceContext available for session")
		}

		util.AddPushChan(s)
		//sessionId, _ := s.Get("id")
		//logging.LogInfof("ws [%s] connected", sessionId)
	})

	util.WebSocketServer.HandleDisconnect(func(s *melody.Session) {
		util.RemovePushChan(s)
		//sessionId, _ := s.Get("id")
		//logging.LogInfof("ws [%s] disconnected", sessionId)
	})

	util.WebSocketServer.HandleError(func(s *melody.Session, err error) {
		//sessionId, _ := s.Get("id")
		//logging.LogWarnf("ws [%s] failed: %s", sessionId, err)
	})

	util.WebSocketServer.HandleClose(func(s *melody.Session, i int, str string) error {
		//sessionId, _ := s.Get("id")
		//logging.LogDebugf("ws [%s] closed: %v, %v", sessionId, i, str)
		return nil
	})

	util.WebSocketServer.HandleMessage(func(s *melody.Session, msg []byte) {
		start := time.Now()
		logging.LogTracef("request [%s]", shortReqMsg(msg))
		request := map[string]interface{}{}
		if err := gulu.JSON.UnmarshalJSON(msg, &request); err != nil {
			result := util.NewResult()
			result.Code = -1
			result.Msg = "Bad Request"
			responseData, _ := gulu.JSON.MarshalJSON(result)
			s.Write(responseData)
			return
		}

		if _, ok := s.Get("app"); !ok {
			result := util.NewResult()
			result.Code = -1
			result.Msg = "Bad Request"
			s.Write(result.Bytes())
			return
		}

		cmdStr := request["cmd"].(string)
		cmdId := request["reqId"].(float64)
		param := request["param"].(map[string]interface{})
		command := cmd.NewCommand(cmdStr, cmdId, param, s)
		if nil == command {
			result := util.NewResult()
			result.Code = -1
			result.Msg = "can not find command [" + cmdStr + "]"
			s.Write(result.Bytes())
			return
		}
		if !command.IsRead() {
			readonly := util.ReadOnly
			if !readonly {
				if token := model.ParseXAuthToken(s.Request); token != nil {
					readonly = token.Valid && model.IsValidRole(model.GetClaimRole(model.GetTokenClaims(token)), []model.Role{
						model.RoleReader,
						model.RoleVisitor,
					})
				}
			}

			if readonly {
				result := util.NewResult()
				result.Code = -1
				result.Msg = model.Conf.Language(34)
				s.Write(result.Bytes())
				return
			}
		}

		end := time.Now()
		logging.LogTracef("parse cmd [%s] consumed [%d]ms", command.Name(), end.Sub(start).Milliseconds())

		cmd.Exec(command)
	})
}

func serveWebDAV(ginServer *gin.Engine) {
	// REF: https://github.com/fungaren/gin-webdav
	handler := webdav.Handler{
		Prefix:     "/webdav/",
		FileSystem: webdav.Dir(util.WorkspaceDir),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if nil != err {
				logging.LogErrorf("WebDAV [%s %s]: %s", r.Method, r.URL.String(), err.Error())
			}
			// logging.LogDebugf("WebDAV [%s %s]", r.Method, r.URL.String())
		},
	}

	ginGroup := ginServer.Group("/webdav", model.CheckAuth, model.CheckAdminRole)
	// ginGroup.Any NOT support extension methods (PROPFIND etc.)
	ginGroup.Match(WebDavMethods, "/*path", func(c *gin.Context) {
		if util.ReadOnly {
			switch c.Request.Method {
			case http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				MethodMkCol,
				MethodCopy,
				MethodMove,
				MethodLock,
				MethodUnlock,
				MethodPropPatch:
				c.AbortWithError(http.StatusForbidden, fmt.Errorf(model.Conf.Language(34)))
				return
			}
		}
		handler.ServeHTTP(c.Writer, c.Request)
	})
}

func serveCalDAV(ginServer *gin.Engine) {
	// REF: https://github.com/emersion/hydroxide/blob/master/carddav/carddav.go
	handler := caldav.Handler{
		Backend: &model.CalDavBackend{},
		Prefix:  model.CalDavPrincipalsPath,
	}

	ginServer.Match(CalDavMethods, "/.well-known/caldav", func(c *gin.Context) {
		// logging.LogDebugf("CalDAV -> [%s] %s", c.Request.Method, c.Request.URL.String())
		handler.ServeHTTP(c.Writer, c.Request)
	})

	ginGroup := ginServer.Group(model.CalDavPrefixPath, model.CheckAuth, model.CheckAdminRole)
	ginGroup.Match(CalDavMethods, "/*path", func(c *gin.Context) {
		// logging.LogDebugf("CalDAV -> [%s] %s", c.Request.Method, c.Request.URL.String())
		if util.ReadOnly {
			switch c.Request.Method {
			case http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				MethodMkCol,
				MethodCopy,
				MethodMove,
				MethodLock,
				MethodUnlock,
				MethodPropPatch:
				c.AbortWithError(http.StatusForbidden, fmt.Errorf(model.Conf.Language(34)))
				return
			}
		}
		handler.ServeHTTP(c.Writer, c.Request)
		// logging.LogDebugf("CalDAV <- [%s] %v", c.Request.Method, c.Writer.Status())
	})
}

func serveCardDAV(ginServer *gin.Engine) {
	// REF: https://github.com/emersion/hydroxide/blob/master/carddav/carddav.go
	handler := carddav.Handler{
		Backend: &model.CardDavBackend{},
		Prefix:  model.CardDavPrincipalsPath,
	}

	ginServer.Match(CardDavMethods, "/.well-known/carddav", func(c *gin.Context) {
		// logging.LogDebugf("CardDAV [/.well-known/carddav]")
		handler.ServeHTTP(c.Writer, c.Request)
	})

	ginGroup := ginServer.Group(model.CardDavPrefixPath, model.CheckAuth, model.CheckAdminRole)
	ginGroup.Match(CardDavMethods, "/*path", func(c *gin.Context) {
		if util.ReadOnly {
			switch c.Request.Method {
			case http.MethodPost,
				http.MethodPut,
				http.MethodDelete,
				MethodMkCol,
				MethodCopy,
				MethodMove,
				MethodLock,
				MethodUnlock,
				MethodPropPatch:
				c.AbortWithError(http.StatusForbidden, fmt.Errorf(model.Conf.Language(34)))
				return
			}
		}
		// TODO: Can't handle Thunderbird's PROPFIND request with prop <current-user-privilege-set/>
		handler.ServeHTTP(c.Writer, c.Request)
		// logging.LogDebugf("CardDAV <- [%s] %v", c.Request.Method, c.Writer.Status())
	})
}

func shortReqMsg(msg []byte) []byte {
	s := gulu.Str.FromBytes(msg)
	max := 128
	if len(s) > max {
		count := 0
		for i := range s {
			count++
			if count > max {
				return gulu.Str.ToBytes(s[:i] + "...")
			}
		}
	}
	return msg
}

func corsMiddleware() gin.HandlerFunc {
	allowMethods := strings.Join(HttpMethods, ", ")
	allowWebDavMethods := strings.Join(WebDavMethods, ", ")
	allowCalDavMethods := strings.Join(CalDavMethods, ", ")
	allowCardDavMethods := strings.Join(CardDavMethods, ", ")

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "origin, Content-Length, Content-Type, Authorization")
		c.Header("Access-Control-Allow-Private-Network", "true")

		if strings.HasPrefix(c.Request.RequestURI, "/webdav") {
			c.Header("Access-Control-Allow-Methods", allowWebDavMethods)
			c.Next()
			return
		}

		if strings.HasPrefix(c.Request.RequestURI, "/caldav") {
			c.Header("Access-Control-Allow-Methods", allowCalDavMethods)
			c.Next()
			return
		}

		if strings.HasPrefix(c.Request.RequestURI, "/carddav") {
			c.Header("Access-Control-Allow-Methods", allowCardDavMethods)
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Methods", allowMethods)

		switch c.Request.Method {
		case http.MethodOptions:
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// jwtMiddleware is a middleware to check jwt token
// REF: https://github.com/siyuan-note/siyuan/issues/11364
func jwtMiddleware(c *gin.Context) {
	if token := model.ParseXAuthToken(c.Request); token != nil {
		// c.Request.Header.Del(model.XAuthTokenKey)
		if token.Valid {
			claims := model.GetTokenClaims(token)
			c.Set(model.ClaimsContextKey, claims)
			c.Set(model.RoleContextKey, model.GetClaimRole(claims))
			c.Next()
			return
		}
	}
	c.Set(model.RoleContextKey, model.RoleVisitor)
	c.Next()
	return
}

func serveFixedStaticFiles(ginServer *gin.Engine) {
	ginServer.StaticFile("favicon.ico", filepath.Join(util.WorkingDir, "stage", "icon.png"))

	ginServer.StaticFile("manifest.json", filepath.Join(util.WorkingDir, "stage", "manifest.webmanifest"))
	ginServer.StaticFile("manifest.webmanifest", filepath.Join(util.WorkingDir, "stage", "manifest.webmanifest"))

	ginServer.StaticFile("service-worker.js", filepath.Join(util.WorkingDir, "stage", "service-worker.js"))
}
