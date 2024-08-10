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
	// 静态资源
	embedFS embed.FS

	// 设置embed文件到静态资源的相对路径，也就是embed注释里的路径
	path string
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
	engine.Use(middleware.DbCheck(), middleware.NetFilter())

	engine.NoRoute(func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	// 不需要认证的路由
	var open = engine.Group("")
	open.StaticFS("/app", http.FS(StaticFile{embedFS: dir, path: "webroot"}))
	open.POST("/api/login", service.UserLogin)
	open.POST("/api/sys/db_conn_check", service.DbConnCheck)
	open.GET("/api/sys/is_init", service.GetIsInit)
	open.POST("/api/sys/init", service.SysInit)

	// 需要认证的路由
	var auth = engine.Group("", middleware.SysInit(), middleware.JWTAuth())

	{ // SSH 连接配置
		auth.GET("/api/conn_conf", service.ConfFindAll)
		auth.GET("/api/conn_conf/:id", service.ConfFindByID)
		auth.POST("/api/conn_conf", service.ConfCreate)
		auth.PUT("/api/conn_conf", service.ConfUpdateById)
		auth.DELETE("/api/conn_conf/:id", service.ConfDeleteById)
	}

	{ // 命令收藏
		auth.GET("/api/cmd_note", service.CmdNoteFindAll)
		auth.GET("/api/cmd_note/:id", service.CmdNoteFindByID)
		auth.POST("/api/cmd_note", service.CmdNoteCreate)
		auth.PUT("/api/cmd_note", service.CmdNoteUpdateById)
		auth.DELETE("/api/cmd_note/:id", service.CmdNoteDeleteById)
	}

	{ // 策略配置
		auth.GET("/api/policy_conf", service.PolicyConfFindAll)
		auth.GET("/api/policy_conf/:id", service.PolicyConfFindByID)
		auth.POST("/api/policy_conf", service.PolicyConfCreate)
		auth.PUT("/api/policy_conf", service.PolicyConfUpdateById)
		auth.DELETE("/api/policy_conf/:id", service.PolicyConfDeleteById)
	}

	{ // 访问控制
		auth.GET("/api/net_filter", service.NetFilterFindAll)
		auth.GET("/api/net_filter/:id", service.NetFilterFindByID)
		auth.POST("/api/net_filter", service.NetFilterCreate)
		auth.PUT("/api/net_filter", service.NetFilterUpdateById)
		auth.DELETE("/api/net_filter/:id", service.NetFilterDeleteById)
	}

	{ // 用户管理
		auth.GET("/api/user", service.UserFindAll)
		auth.GET("/api/user/:id", service.UserFindByID)
		auth.POST("/api/user", service.UserCreate)
		auth.PUT("/api/user", service.UserUpdateById)
		auth.DELETE("/api/user/:id", service.UserDeleteById)
		auth.PATCH("/api/user/check_name_exists", service.CheckUserNameExists)
		auth.PATCH("/api/user/pwd", service.ModifyPasswd)
	}

	{ // 审计日志
		auth.POST("/api/login_audit", service.LoginAuditSearch)
	}

	{ // SSH链接
		auth.GET("/api/conn_manage/online_client", service.GetOnlineClient)
		auth.PUT("/api/conn_manage/refresh_conn_time", service.RefreshConnTime)
		auth.POST("/api/sftp/create_dir", service.SftpCreateDir)
		auth.POST("/api/sftp/list", service.SftpList)
		auth.GET("/api/sftp/download", service.SftpDownLoad)
		auth.PUT("/api/sftp/upload", service.SftpUpload)
		auth.DELETE("/api/sftp/delete", service.SftpDelete)
		auth.GET("/api/ssh/conn", service.NewSshConn)
		auth.PATCH("/api/ssh/conn", service.ResizeWindow)
		auth.POST("/api/ssh/exec", service.ExecCommand)
		auth.POST("/api/ssh/disconnect", service.Disconnect)
		auth.POST("/api/ssh/create_session", service.CreateSessionId)
	}

	{ // 系统配置
		auth.GET("/api/sys/config", service.GetRunConf)
		auth.POST("/api/sys/config", service.SetRunConf)
	}

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
