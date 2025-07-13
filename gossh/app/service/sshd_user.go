package service

import (
	"gossh/app/model"
	"gossh/gin"
	"log/slog"
	"strconv"
)

func SshdUserCreate(c *gin.Context) {
	var sshdUser model.SshdUser
	if err := c.ShouldBind(&sshdUser); err != nil {
		slog.Error("UserCreate 绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}

	err := sshdUser.Create(&sshdUser)
	if err != nil {
		slog.Error("创建用户错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "创建用户错误"})
		return
	}
	SshdUserFindAll(c)
}

func CheckSshdUserNameExists(c *gin.Context) {
	type Name struct {
		Name string `form:"name" binding:"required,min=1,max=128" json:"name"`
	}
	var name Name
	if err := c.ShouldBind(&name); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}

	var sshdUser model.SshdUser
	tmp, err := sshdUser.FindByName(name.Name)
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

func SshdUserFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	var sshdUser model.SshdUser
	data, err := sshdUser.FindByID(uint(id))
	if err != nil {
		slog.Error("FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取用户信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func SshdUserFindAll(c *gin.Context) {
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

	var sshdUser model.SshdUser
	data, err := sshdUser.FindAll(limit, offset)
	if err != nil {
		slog.Error("user.FindAl错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 4, "msg": "获取用户信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func SshdUserUpdateById(c *gin.Context) {
	var sshdUser model.SshdUser
	if err := c.ShouldBind(&sshdUser); err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	err := sshdUser.UpdateById(sshdUser.ID, &sshdUser)
	if err != nil {
		slog.Error("UpdateById错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "更新用户错误"})
		return
	}
	SshdUserFindAll(c)
}

func SshdUserDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	var sshdUser model.SshdUser
	err = sshdUser.DeleteByID(uint(id))
	if err != nil {
		slog.Error("user.DeleteByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "删除用户错误"})
		return
	}
	SshdUserFindAll(c)
}
