package middleware

import (
	"errors"
	"gossh/app/config"
	"gossh/gin"
	"gossh/gin/jwt"
	"strings"
	"time"
)

// JwtSecret 生成JWT签名的密钥
var JwtSecret = []byte(config.DefaultConfig.JwtSecret)

type JwtClaims struct {
	// 用户Id
	Id uint

	// 标准Claims结构体，可设置8个标准字段
	jwt.RegisteredClaims
}

// GenerateToken 登录成功后调用，传入SshUser结构体
func GenerateToken(id uint) (string, error) {

	// 两个小时有效期
	expirationTime := time.Now().Add(config.DefaultConfig.JwtExpire)
	claims := &JwtClaims{
		Id: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "go_web_ssh",
		},
	}
	// 生成Token，指定签名算法和claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名
	tokenString, err := token.SignedString(JwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ParseToken(tokenString string) (*JwtClaims, error) {
	claims := &JwtClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		return JwtSecret, nil
	})
	// 若token只是过期claims是有数据的，若token无法解析claims无数据
	return claims, err
}

func RenewToken(claims *JwtClaims) (string, error) {
	// 若token过期不超过10分钟则给它续签
	if withinLimit(claims.ExpiresAt.Time.Unix(), 600) {
		return GenerateToken(claims.Id)
	}
	return "", errors.New("登录已过期")
}

// 计算过期时间是否超过l
func withinLimit(s, l int64) bool {
	return time.Now().Unix()-s < l
}

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.Request.Header.Get("Authorization")
		if len(auth) == 0 {
			// Websocket SSE 从请求参数取
			auth = c.Query("Authorization")
		}
		if len(auth) == 0 {
			// 无token直接拒绝
			c.Abort()
			c.JSON(401, gin.H{"code": 401, "msg": "请添加Authorization请求头"})
			return
		}
		// 校验token
		claims, err := ParseToken(auth)
		if err != nil {
			if strings.Contains(err.Error(), "token is expired") {
				// 若过期，调用续签函数
				newToken, _ := RenewToken(claims)
				if newToken != "" {
					// 续签成功返回头设置一个NewToken字段
					c.Header("NewToken", newToken)
					//c.Request.Header.Set("Authorization", newToken)
					c.Set("uid", claims.Id)
					c.Next()
					return
				}
			}
			// Token验证失败或续签失败直接拒绝请求
			c.Abort()
			c.JSON(401, gin.H{"code": 401, "msg": "未登录"})
			return
		}
		c.Set("uid", claims.Id)
		// token未过期继续执行其他中间件
		c.Next()
	}
}
