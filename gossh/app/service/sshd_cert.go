package service

import (
	"gossh/app/model"
	"gossh/gin"
	"log/slog"
	"strconv"
)

func SshdCertCreate(c *gin.Context) {
	var sshdCert model.SshdCert
	if err := c.ShouldBind(&sshdCert); err != nil {
		slog.Error("SshdCertCreate 绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}

	err := sshdCert.Create(&sshdCert)
	if err != nil {
		slog.Error("创建SSHD证书错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "创建SSHD证书错误"})
		return
	}
	SshdCertFindAll(c)
}

func CheckSshdCertNameExists(c *gin.Context) {
	type Name struct {
		Name string `form:"name" binding:"required,min=1,max=128" json:"name"`
	}
	var name Name
	if err := c.ShouldBind(&name); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}

	var sshdCert model.SshdCert
	tmp, err := sshdCert.FindByName(name.Name)
	if err != nil {
		slog.Error("FindByName错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取SSHD证书信息错误"})
		return
	}

	if tmp.ID != 0 {
		c.JSON(200, gin.H{"code": 4, "msg": "名称已经存存在"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok"})
	return
}

func SshdCertFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	var sshdCert model.SshdCert
	data, err := sshdCert.FindByID(uint(id))
	if err != nil {
		slog.Error("FindByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "获取SSHD证书信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func SshdCertFindAll(c *gin.Context) {
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

	var sshdCert model.SshdCert
	data, err := sshdCert.FindAll(limit, offset)
	if err != nil {
		slog.Error("sshdCert.FindAll错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 4, "msg": "获取SSHD证书信息错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func SshdCertUpdateById(c *gin.Context) {
	var sshdCert model.SshdCert
	if err := c.ShouldBind(&sshdCert); err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	err := sshdCert.UpdateById(sshdCert.ID, &sshdCert)
	if err != nil {
		slog.Error("UpdateById错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "更新SSHD证书错误"})
		return
	}
	SshdCertFindAll(c)
}

func SshdCertDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		slog.Error("获取ID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取ID错误"})
		return
	}

	var sshdCert model.SshdCert
	err = sshdCert.DeleteByID(uint(id))
	if err != nil {
		slog.Error("sshdCert.DeleteByID错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "删除SSHD证书错误"})
		return
	}
	SshdCertFindAll(c)
}

func GetSshdCertAuthorizedKeys(c *gin.Context) {
	var sshdCert model.SshdCert
	text, err := sshdCert.GetAuthorizedKeys()
	if err != nil {
		slog.Error("GetAuthorizedKeys failed", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "获取SSHD证书错误", "err_msg": err})
		return
	}
	c.JSON(200, gin.H{"code": 5, "msg": "ok", "data": text})
}
