package middleware

import (
	"fmt"
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
	"log/slog"
	"net"
)

func check(ip net.IP) bool {
	var policyConf model.PolicyConf
	conf, err := policyConf.FindByID(1)
	if err != nil {
		slog.Error("get policyConf:", "err_msg", err.Error())
		return false
	}
	// 白名单检查
	if conf.NetPolicy == "Y" {
		// slog.Info("netFilterPolicy", "value", "Y")
		var filter model.NetFilter
		list, err := filter.FindAllPolicy("Y")
		if err != nil {
			slog.Error("FindAllPolicy:", "err_msg", err.Error())
		}
		isOK := false
		for _, item := range list {
			_, ipNet, err := net.ParseCIDR(item.Cidr)
			if err != nil {
				slog.Error("net.ParseCIDR:", "err_msg", err.Error())
				return false
			}
			if ipNet.Contains(ip) {
				isOK = true
				break
			}
		}
		return isOK
	}

	// 黑名单检查
	if conf.NetPolicy == "N" {
		// slog.Info("netFilterPolicy", "value", "N")
		var filter model.NetFilter
		list, err := filter.FindAllPolicy("N")
		if err != nil {
			slog.Error("FindAllPolicy:", "err_msg", err.Error())
		}
		isOK := true
		for _, item := range list {
			_, ipNet, err := net.ParseCIDR(item.Cidr)
			if err != nil {
				slog.Error("net.ParseCIDR:", "err_msg", err.Error())
				return false
			}
			if ipNet.Contains(ip) {
				isOK = false
				break
			}
		}
		return isOK
	}
	return false
}

func NetFilter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 系统没有进行初始化,过滤功能不生效
		if !config.DefaultConfig.IsInit {
			c.Next()
			return
		}

		ip := net.ParseIP(c.RemoteIP())
		if !check(ip) {
			c.JSON(403, gin.H{"err_msg": fmt.Sprintf("Deny %s", ip.String())})
			c.Abort()
			return
		}
		c.Next()
	}
}
