package middleware

import (
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
)

func DbCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.DefaultConfig.IsInit {
			c.Next()
			return
		}
		if model.Db != nil {
			tx := model.Db.Exec("select 1=1")
			if tx.Error == nil {
				c.Next()
			} else {
				err := model.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn)
				if err != nil {
					c.Abort()
					c.JSON(500, gin.H{"code": 500, "msg": "数据库连接错误:" + err.Error()})
					return
				}
			}
		}
		c.Next()
	}
}
