package service

import (
	"gossh/app/model"
	"gossh/gin"
	"strconv"
)

func NetFilterCreate(c *gin.Context) {
	var netFilter model.NetFilter
	if err := c.ShouldBind(&netFilter); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	err := netFilter.Create(&netFilter)
	if err != nil {
		c.JSON(200, gin.H{"code": 3, "msg": err.Error()})
		return
	}
	NetFilterFindAll(c)
}

func NetFilterFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var netFilter model.NetFilter
	data, err := netFilter.FindByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func NetFilterFindAll(c *gin.Context) {
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

	var netFilter model.NetFilter
	data, err := netFilter.FindAll(offset, limit)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func NetFilterUpdateById(c *gin.Context) {
	var netFilter model.NetFilter
	if err := c.ShouldBind(&netFilter); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := netFilter.UpdateById(netFilter.ID, &netFilter)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	NetFilterFindAll(c)
}

func NetFilterDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var netFilter model.NetFilter
	err = netFilter.DeleteByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	NetFilterFindAll(c)
}
