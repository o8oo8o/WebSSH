package middleware

import (
	"gossh/app/config"
	"gossh/gin"
)

func SysInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.DefaultConfig.IsInit {
			// 需要进行系统初始化
			c.Abort()
			c.JSON(401, gin.H{"code": 401, "msg": "请对系统进行初始化"})
			return
		}
		c.Next()
	}
}
