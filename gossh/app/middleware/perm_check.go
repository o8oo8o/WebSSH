package middleware

import (
	"fmt"
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
	"slices"
	"strings"
)

var checkRoute = []string{
	// "GET:/app/*filepath",
	// "GET:/web_base_dir",
	// "HEAD:/app/*filepath",

	// "GET:/api/ssh/conn",
	// "PATCH:/api/ssh/conn",
	// "POST:/api/ssh/exec",
	// "POST:/api/ssh/disconnect",
	// "POST:/api/ssh/create_session",

	// "GET:/api/conn_conf",
	// "GET:/api/conn_conf/:id",
	// "POST:/api/conn_conf",
	// "PUT:/api/conn_conf",
	// "DELETE:/api/conn_conf/:id",

	// "GET:/api/cmd_note",
	// "GET:/api/cmd_note/:id",
	// "POST:/api/cmd_note",
	// "PUT:/api/cmd_note",
	// "DELETE:/api/cmd_note/:id",

	// "GET:/api/sftp/download",
	// "POST:/api/sftp/create_dir",
	// "POST:/api/sftp/list",
	// "DELETE:/api/sftp/delete",
	// "PUT:/api/sftp/upload",

	// "PUT:/api/conn_manage/refresh_conn_time",
	// "POST:/api/login",
	// "PATCH:/api/user/pwd",

	"GET:/api/sshd_cert",
	"GET:/api/sshd_cert_text",
	"GET:/api/sshd_cert/:id",
	"GET:/api/sshd_user",
	"GET:/api/sshd_user/:id",
	"GET:/api/sys/is_init",
	"GET:/api/sys/config",
	"GET:/api/conn_manage/online_client",
	"GET:/api/policy_conf",
	"GET:/api/policy_conf/:id",
	"GET:/api/net_filter",
	"GET:/api/net_filter/:id",
	"GET:/api/user",
	"GET:/api/user/:id",
	"POST:/api/sshd_user",
	"POST:/api/sshd_cert",
	"POST:/api/sys/db_conn_check",
	"POST:/api/sys/init",
	"POST:/api/sys/config",
	"POST:/api/login_audit",
	"POST:/api/policy_conf",
	"POST:/api/net_filter",
	"POST:/api/user",
	"PUT:/api/sshd_user",
	"PUT:/api/sshd_cert",
	"PUT:/api/policy_conf",
	"PUT:/api/net_filter",
	"PUT:/api/user",
	"DELETE:/api/sshd_user/:id",
	"DELETE:/api/sshd_cert/:id",
	"DELETE:/api/policy_conf/:id",
	"DELETE:/api/net_filter/:id",
	"DELETE:/api/user/:id",
	"PATCH:/api/sshd_user/check_name_exists",
	"PATCH:/api/sshd_cert/check_name_exists",
	"PATCH:/api/user/check_name_exists",
}

func PremCheck(engine *gin.Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
			for _, route := range engine.Routes() {
				path := route.Path
				method := route.Method
				fmt.Printf("Route==>: %s:%s\n", method, path)
			}
		*/
		var tmp = c.FullPath()
		var baseDir = strings.Trim(config.DefaultConfig.WebBaseDir, " ")
		if baseDir != "" && strings.HasPrefix(tmp, baseDir) {
			tmp = strings.TrimPrefix(tmp, baseDir)
		}

		full := fmt.Sprintf("%s:%s", c.Request.Method, tmp)
		// slog.Info("PermCheck", "base", baseDir, "path", full)
		if slices.Contains(checkRoute, full) {
			var wu model.WebUser
			u, err := wu.FindByID(c.GetUint("uid"))
			if err != nil || u.IsAdmin == "N" {
				c.JSON(200, gin.H{"code": 403, "msg": "权限检查失败,非管理员拒绝操作"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
