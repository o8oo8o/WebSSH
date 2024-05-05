package service

import (
	"gossh/app/model"
	"gossh/gin"
	"strconv"
)

func PolicyConfCreate(c *gin.Context) {
	var conf model.PolicyConf
	if err := c.ShouldBind(&conf); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	err := conf.Create(&conf)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error()})
		return
	}
	PolicyConfFindAll(c)
}

func PolicyConfFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var conf model.PolicyConf
	data, err := conf.FindByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func PolicyConfFindAll(c *gin.Context) {
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

	var conf model.PolicyConf
	data, err := conf.FindAll(offset, limit)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func PolicyConfUpdateById(c *gin.Context) {
	var conf model.PolicyConf
	if err := c.ShouldBind(&conf); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := conf.UpdateById(conf.ID, &conf)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}

	data, err := conf.FindByID(conf.ID)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func PolicyConfDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var conf model.PolicyConf
	err = conf.DeleteByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	PolicyConfFindAll(c)
}
