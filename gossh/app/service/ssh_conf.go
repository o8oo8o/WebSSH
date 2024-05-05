package service

import (
	"gossh/app/model"
	"gossh/gin"
	"strconv"
)

func ConfCreate(c *gin.Context) {
	var config model.SshConf
	if err := c.ShouldBind(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	config.Uid = c.GetUint("uid")
	err := config.Create(&config)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	ConfFindAll(c)
}

func ConfFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var config model.SshConf
	data, err := config.FindByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func ConfFindAll(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10000"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}

	var config model.SshConf
	data, err := config.FindAll(offset, limit, c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func ConfUpdateById(c *gin.Context) {
	var config model.SshConf
	if err := c.ShouldBind(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := config.UpdateById(config.ID, c.GetUint("uid"), &config)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	// c.JSON(200, gin.H{"code": 0, "msg": "ok"})
	ConfFindAll(c)
}

func ConfDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var config model.SshConf
	err = config.DeleteByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	// c.JSON(200, gin.H{"code": 0, "msg": "ok"})
	ConfFindAll(c)
}
