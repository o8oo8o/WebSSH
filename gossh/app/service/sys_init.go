package service

import (
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
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
	var user = model.SshUser{
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

	var policyConf = model.PolicyConf{
		NetPolicy: "N",
	}
	err = policyConf.Create(&policyConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error(), "data": "初始化网络策略错误"})
		return
	}

	// 5.覆盖默认配置
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
	c.JSON(200, gin.H{"code": 0, "msg": "系统初始化完成"})
}
