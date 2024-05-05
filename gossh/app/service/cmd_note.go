package service

import (
	"gossh/app/model"
	"gossh/gin"
	"strconv"
)

func CmdNoteCreate(c *gin.Context) {
	var cmd model.CmdNote
	if err := c.ShouldBind(&cmd); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	cmd.Uid = c.GetUint("uid")
	err := cmd.Create(&cmd)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}

func CmdNoteFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var cmd model.CmdNote
	data, err := cmd.FindByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func CmdNoteFindAll(c *gin.Context) {
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

	var cmd model.CmdNote
	data, err := cmd.FindAll(offset, limit, c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func CmdNoteUpdateById(c *gin.Context) {
	var cmd model.CmdNote
	if err := c.ShouldBind(&cmd); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := cmd.UpdateById(cmd.ID, c.GetUint("uid"), &cmd)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}

func CmdNoteDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var cmd model.CmdNote
	err = cmd.DeleteByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}
