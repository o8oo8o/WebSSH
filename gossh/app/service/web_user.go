package service

import (
	"gossh/app/config"
	"gossh/app/middleware"
	"gossh/app/model"
	"gossh/app/utils"
	"gossh/gin"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func UserCreate(c *gin.Context) {
	var user model.WebUser
	if err := c.ShouldBind(&user); err != nil {
		slog.Error("UserCreate 绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	var err error
	user.IsRoot = "N"
	err = user.Create(&user)
	if err != nil {
		slog.Error("创建用户错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "创建用户错误"})
		return
	}
	UserFindAll(c)
}

func ModifyPasswd(c *gin.Context) {
	type password struct {
		Pwd string `form:"pwd" binding:"required,min=1,max=64" json:"pwd"`
	}

	var pwd password
	if err := c.ShouldBind(&pwd); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}

	uid := c.GetUint("uid")
	var tmp model.WebUser
	user, err := tmp.FindByID(uid)
	if err != nil {
		slog.Error("FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": "获取用户信息错误"})
		return
	}

	user.Pwd = pwd.Pwd
	err = user.UpdatePassword(uid, &user)
	if err != nil {
		slog.Error("UpdatePassword错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "更新用户密码错误"})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "更新密码成功"})
}

func CheckUserNameExists(c *gin.Context) {
	type Name struct {
		Name string `form:"name" binding:"required,min=1,max=128" json:"name"`
	}
	var name Name
	if err := c.ShouldBind(&name); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	var user model.WebUser

	tmp, err := user.FindByName(name.Name)
	if err != nil {
		slog.Error("FindByName错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取用户信息错误"})
		return
	}

	if tmp.ID != 0 {
		c.JSON(200, gin.H{"code": 4, "msg": "用户名已经存存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok"})
	return
}

func UserFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}
	var user model.WebUser
	data, err := user.FindByID(uint(id))
	if err != nil {
		slog.Error("FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取用户信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func UserFindAll(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10000"))
	if err != nil {
		slog.Error("获取limit错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取limit错误"})
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		slog.Error("获取offset错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": "获取offset错误"})
		return
	}

	var user model.WebUser
	data, err := user.FindAll(limit, offset)
	if err != nil {
		slog.Error("user.FindAl错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 4, "msg": "获取用户信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func UserUpdateById(c *gin.Context) {
	var user model.WebUser
	if err := c.ShouldBind(&user); err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	tmpUser, err := user.FindByID(user.ID)
	if err != nil {
		slog.Error("FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取用户信息错误"})
		return
	}

	// 防止越权操作
	if tmpUser.IsRoot == "N" {
		user.IsRoot = "N"
	}

	if tmpUser.IsRoot == "Y" {
		c.JSON(200, gin.H{"code": 4, "msg": "内置Root用户不能更新"})
		return
	}

	err = user.UpdateById(user.ID, &user)
	if err != nil {
		slog.Error("UpdateById错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "更新用户错误"})
		return
	}
	UserFindAll(c)
}

func UserDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}
	var user model.WebUser
	tmpUser, err := user.FindByID(uint(id))
	if err != nil {
		slog.Error("user.FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取用户信息错误"})
		return
	}

	// 防止越权操作
	if tmpUser.IsRoot == "Y" {
		c.JSON(200, gin.H{"code": 4, "msg": "内置Root用户不能删除"})
		return
	}

	err = user.DeleteByID(uint(id))
	if err != nil {
		slog.Error("user.DeleteByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "删除用户错误"})
		return
	}
	UserFindAll(c)
}

func UserLogin(c *gin.Context) {
	if !config.DefaultConfig.IsInit {
		slog.Warn("system no init")
		c.JSON(401, gin.H{"code": 401, "msg": "请对系统进行初始化"})
		return
	}
	type Param struct {
		Name string `form:"name" binding:"required,min=1,max=64" json:"name"`
		Pwd  string `form:"pwd" binding:"required,min=1,max=64" json:"pwd"`
	}

	var loginAudit model.LoginAudit
	var param Param

	audit := model.LoginAudit{
		ClientIp:  c.ClientIP(),
		UserAgent: utils.TruncateString(c.Request.UserAgent(), 500),
		ErrMsg:    "请求参数错误",
		IsSuccess: "N",
		OccurAt:   model.DateTime(time.Now()),
	}
	if err := c.ShouldBind(&param); err != nil {
		audit.Name = utils.TruncateString(param.Name, 60)
		audit.Pwd = utils.TruncateString(param.Pwd, 60)
		_ = loginAudit.Create(&audit)
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	audit.Name = utils.TruncateString(param.Name, 60)
	audit.Pwd = utils.TruncateString(param.Pwd, 60)

	var user model.WebUser
	u, err := user.FindByNameAndPwd(param.Name, param.Pwd)
	if err != nil {
		audit.ErrMsg = "账号密码错误"
		_ = loginAudit.Create(&audit)
		slog.Error("账号密码错误", "err_msg", err.Error())
		c.JSON(401, gin.H{"code": 2, "msg": "账号密码错误"})
		return
	}

	if u.IsEnable == "N" {
		audit.ErrMsg = "账号已禁用"
		_ = loginAudit.Create(&audit)
		c.JSON(401, gin.H{"code": 3, "msg": "账号已禁用"})
		return
	}

	if u.ExpiryAt.ToTime().Unix() < time.Now().Unix() {
		audit.ErrMsg = "账号已过期"
		_ = loginAudit.Create(&audit)
		c.JSON(401, gin.H{"code": 4, "msg": "账号已过期"})
		return
	}

	tokenString, err := middleware.GenerateToken(u.ID)
	if err != nil {
		audit.ErrMsg = "生成Token错误"
		_ = loginAudit.Create(&audit)
		c.JSON(401, gin.H{"code": 5, "msg": err.Error()})
		return
	}

	audit.Name = param.Name
	audit.Pwd = "*"
	audit.ErrMsg = "*"
	audit.IsSuccess = "Y"
	_ = loginAudit.Create(&audit)
	c.JSON(http.StatusOK, gin.H{
		"code":           0,
		"token":          tokenString,
		"msg":            "登录成功",
		"is_root":        u.IsRoot,
		"is_admin":       u.IsAdmin,
		"user_name":      u.Name,
		"user_desc":      u.DescInfo,
		"user_expiry_at": u.ExpiryAt.String(),
	})
}
