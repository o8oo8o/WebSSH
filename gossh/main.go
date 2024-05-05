package main

import (
	"embed"
	"errors"
	"fmt"
	"gossh/app/config"
	"gossh/app/middleware"
	"gossh/app/service"
	"gossh/gin"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// 使用go 1.16+ 新特性
//
//go:embed webroot
var dir embed.FS

// StaticFile 嵌入普通的静态资源
type StaticFile struct {
	embedFS embed.FS // 静态资源
	path    string   // 设置embed文件到静态资源的相对路径，也就是embed注释里的路径
}

// Open 静态资源被访问的核心逻辑
func (w StaticFile) Open(name string) (fs.File, error) {
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errors.New("http: invalid character in file path")
	}

	fullName := filepath.Join(w.path, filepath.FromSlash(path.Clean("/"+name)))
	fullName = strings.ReplaceAll(fullName, `\`, `/`)
	file, err := w.embedFS.Open(fullName)
	return file, err
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	var engine = gin.Default()
	engine.Use(middleware.NetFilter())

	engine.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	engine.POST("/api/login", service.UserLogin)
	engine.POST("/api/sys/db_conn_check", service.DbConnCheck)
	engine.GET("/api/sys/is_init", service.GetIsInit)
	engine.POST("/api/sys/init", service.SysInit)

	var router = engine.Group("", middleware.SysInit(), middleware.JWTAuth())

	{ // SSH 连接配置
		router.GET("/api/conn_conf", service.ConfFindAll)
		router.GET("/api/conn_conf/:id", service.ConfFindByID)
		router.POST("/api/conn_conf", service.ConfCreate)
		router.PUT("/api/conn_conf", service.ConfUpdateById)
		router.DELETE("/api/conn_conf/:id", service.ConfDeleteById)
	}

	{ // 命令收藏
		router.GET("/api/cmd_note", service.CmdNoteFindAll)
		router.GET("/api/cmd_note/:id", service.CmdNoteFindByID)
		router.POST("/api/cmd_note", service.CmdNoteCreate)
		router.PUT("/api/cmd_note", service.CmdNoteUpdateById)
		router.DELETE("/api/cmd_note/:id", service.CmdNoteDeleteById)
	}

	{ // 策略配置
		router.GET("/api/policy_conf", service.PolicyConfFindAll)
		router.GET("/api/policy_conf/:id", service.PolicyConfFindByID)
		router.POST("/api/policy_conf", service.PolicyConfCreate)
		router.PUT("/api/policy_conf", service.PolicyConfUpdateById)
		router.DELETE("/api/policy_conf/:id", service.PolicyConfDeleteById)
	}

	{ // 访问控制
		router.GET("/api/net_filter", service.NetFilterFindAll)
		router.GET("/api/net_filter/:id", service.NetFilterFindByID)
		router.POST("/api/net_filter", service.NetFilterCreate)
		router.PUT("/api/net_filter", service.NetFilterUpdateById)
		router.DELETE("/api/net_filter/:id", service.NetFilterDeleteById)
	}

	{ // 用户管理
		router.GET("/api/user", service.UserFindAll)
		router.GET("/api/user/:id", service.UserFindByID)
		router.POST("/api/user", service.UserCreate)
		router.PUT("/api/user", service.UserUpdateById)
		router.DELETE("/api/user/:id", service.UserDeleteById)
		router.PATCH("/api/user/check_name_exists", service.CheckUserNameExists)
		router.PATCH("/api/user/pwd", service.ModifyPasswd)
	}

	{ // 审计日志
		router.POST("/api/login_audit", service.LoginAuditSearch)
	}

	{ // SSH链接
		router.GET("/api/conn_manage/online_client", service.GetOnlineClient)
		router.PUT("/api/conn_manage/refresh_conn_time", service.RefreshConnTime)
		router.POST("/api/sftp/create_dir", service.SftpCreateDir)
		router.POST("/api/sftp/list", service.SftpList)
		router.GET("/api/sftp/download", service.SftpDownLoad)
		router.PUT("/api/sftp/upload", service.SftpUpload)
		router.DELETE("/api/sftp/delete", service.SftpDelete)
		router.GET("/api/ssh/conn", service.NewSshConn)
		router.PATCH("/api/ssh/conn", service.ResizeWindow)
		router.POST("/api/ssh/exec", service.ExecCommand)
		router.POST("/api/ssh/disconnect", service.Disconnect)
		router.POST("/api/ssh/create_session", service.CreateSessionId)
	}

	{ // 系统配置
		router.GET("/api/sys/config", service.GetRunConf)
		router.POST("/api/sys/config", service.SetRunConf)
	}

	// 处理前端静态文件
	engine.StaticFS("/app", http.FS(StaticFile{
		embedFS: dir,
		path:    "webroot",
	}))

	address := fmt.Sprintf("%s:%s", config.DefaultConfig.Address, config.DefaultConfig.Port)

	_, certErr := os.Open(config.DefaultConfig.CertFile)
	_, keyErr := os.Open(config.DefaultConfig.KeyFile)

	// 如果证书和私钥文件存在,就使用https协议,否则使用http协议
	if certErr == nil && keyErr == nil {
		slog.Debug("https_server_start")
		err := engine.RunTLS(address, config.DefaultConfig.CertFile, config.DefaultConfig.KeyFile)
		if err != nil {
			slog.Error("RunServeTLSError:", "msg", err.Error())
			os.Exit(1)
			return
		}
	} else {
		slog.Debug("http_server_start")
		err := engine.Run(address)
		if err != nil {
			slog.Error("RunServeError:", "msg", err.Error())
			os.Exit(1)
			return
		}
	}
}
