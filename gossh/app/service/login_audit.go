package service

import (
	"gossh/app/model"
	"gossh/gin"
	"strconv"
)

func AuditFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var audit model.LoginAudit
	data, err := audit.FindByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func LoginAuditSearch(c *gin.Context) {
	type Param struct {
		OccurBegin model.DateTime `json:"occur_begin"  form:"occur_begin"`
		OccurEnd   model.DateTime `json:"occur_end"  form:"occur_end"`
		Offset     int            `form:"offset" json:"offset" binding:"min=0"`
		Limit      int            `form:"limit" json:"limit" binding:"max=1000"`
		Name       string         `form:"name" binding:"max=64" json:"name"`
		ClientIp   string         `form:"client_ip" binding:"max=128" json:"client_ip"`
		IsSuccess  string         `form:"is_success" binding:"max=1" json:"is_success"`
	}
	var p Param
	if err := c.ShouldBind(&p); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if p.Limit == 0 {
		p.Limit = 100
	}
	var audit model.LoginAudit
	data, count, err := audit.Search(p.IsSuccess, p.Name, p.ClientIp, p.OccurBegin, p.OccurEnd, p.Offset, p.Limit)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data, "count": count})
}

func LoginAuditDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var audit model.LoginAudit
	err = audit.DeleteByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
}
