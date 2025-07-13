package service

import (
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
	"log/slog"
	"os/exec"
	"runtime"
)

func GetRunConf(c *gin.Context) {
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": config.DefaultConfig})
}

func SetRunConf(c *gin.Context) {
	if config.DefaultConfig.IsInit {
		c.JSON(200, gin.H{"code": 1, "msg": "已经初始化"})
		return
	}
	var appConfig config.AppConfig
	if err := c.ShouldBind(&appConfig); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := config.RewriteConfig(appConfig)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": config.DefaultConfig})
}

func GetIsInit(c *gin.Context) {
	c.JSON(200, gin.H{
		"code": 0, "msg": "ok", "data": map[string]any{
			"is_init": config.DefaultConfig.IsInit,
		},
	})
}

type InitConfig struct {
	DbConnConf
	JwtSecret     string `json:"jwt_secret"  binding:"required,min=1,max=128"`
	SessionSecret string `json:"session_secret"  binding:"required,min=1,max=128"`
	Username      string `json:"username"  binding:"required,min=1,max=63"`
	Password      string `json:"password" binding:"required,min=1,max=63"`
	SshdHost      string `json:"sshd_host"  binding:"required,min=1,max=127"`
	SshdPort      uint16 `json:"sshd_port" binding:"required,gte=1,lte=65535"`
	SshdUser      string `json:"sshd_user"  binding:"required,min=1,max=63"`
	SshdPwd       string `json:"sshd_pwd" binding:"required,min=1,max=63"`
}

func SysInit(c *gin.Context) {
	var initConf InitConfig
	if err := c.ShouldBind(&initConf); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 1.检查系统是否已经初始化
	if config.DefaultConfig.IsInit {
		c.JSON(200, gin.H{"code": 1, "msg": "系统已经初始化"})
		return
	}

	// 2.数据库连接检查
	dbConf := DbConnConf{
		DbDsn:  initConf.DbDsn,
		DbType: initConf.DbType,
	}
	err := DbConnTestCheck(dbConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 3.数据库表迁移
	err = model.DbMigrate(initConf.DbType, initConf.DbDsn)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error(), "data": "执行数据库迁移错误"})
		return
	}

	// root 用户过期时间
	dateTime, _ := model.NewDateTime("2099-12-31 00:00:00")

	// 4.创建初始化用户
	var user = model.WebUser{
		ID:       0,
		Name:     initConf.Username,
		Pwd:      initConf.Password,
		DescInfo: "管理员",
		IsAdmin:  "Y",
		IsEnable: "Y",
		IsRoot:   "Y",
		ExpiryAt: dateTime,
	}

	err = user.Create(&user)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化创建账号错误"})
		return
	}

	// 5.设置默认网络策略
	var policyConf = model.PolicyConf{
		NetPolicy: "N",
	}
	err = policyConf.Create(&policyConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化网络策略错误"})
		return
	}

	privateKey, err := generateKey("webssh")
	if err != nil {
		slog.Error("failed to generate private key")
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化生成SSHD私钥错误"})
		return
	}

	var shell = "sh"
	if runtime.GOOS == "windows" {
		shell = "powershell"
	} else {
		sh, err := exec.LookPath("bash")
		if err == nil {
			slog.Info("init find shell is bash\n")
			shell = sh
		}
	}

	// 6.初始化sshd服务端配置
	var sshdConf = model.SshdConf{
		ID:            1,
		Name:          "init",
		Host:          initConf.SshdHost,
		Port:          initConf.SshdPort,
		Shell:         shell,
		KeyFile:       string(privateKey),
		KeySeed:       "webssh",
		KeepAlive:     60,
		LoadEnv:       "Y",
		AuthType:      "all",
		ServerVersion: "SSH-2.0-OpenSSH",
	}
	err = sshdConf.Create(&sshdConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化SSHD服务端配置错误"})
		return
	}

	// 7.初始化sshd服务端账号
	var sshdUser = model.SshdUser{
		ID:       1,
		Name:     initConf.SshdUser,
		Pwd:      initConf.SshdPwd,
		DescInfo: "服务器初始化账号",
		IsEnable: "Y",
		WorkDir:  "",
		ExpiryAt: dateTime,
	}
	err = sshdUser.Create(&sshdUser)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化SSHD服务端账号错误"})
		return
	}

	// 8.覆盖默认配置
	var defConf = config.DefaultConfig
	defConf.IsInit = true
	defConf.DbType = initConf.DbType
	defConf.DbDsn = initConf.DbDsn
	defConf.JwtSecret = initConf.JwtSecret
	defConf.SessionSecret = initConf.SessionSecret
	err = config.RewriteConfig(defConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error(), "data": "写入配置错误"})
		return
	}

	// 9.创建默认连接
	var sshConf = model.SshConf{
		Uid:         1,
		Name:        "self_sshd",
		Address:     "127.0.0.1",
		User:        initConf.SshdUser,
		Pwd:         initConf.SshdPwd,
		AuthType:    "pwd",
		NetType:     "tcp4",
		CertData:    "",
		CertPwd:     "",
		Port:        initConf.SshdPort,
		FontSize:    16,
		Background:  "#000000",
		Foreground:  "#FFFFFF",
		CursorColor: "#FFFFFF",
		FontFamily:  "Courier",
		CursorStyle: "block",
		Shell:       "sh",
		PtyType:     "xterm-256color",
		InitCmd:     "",
		InitBanner:  "# https://github.com/o8oo8o/WebSSH",
	}

	err = sshConf.Create(&sshConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化创建默认连接错误"})
		return
	}

	isStartSshd <- true
	c.JSON(200, gin.H{"code": 0, "msg": "系统初始化完成"})
}
