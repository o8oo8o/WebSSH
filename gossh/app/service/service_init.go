package service

import (
	"gossh/app/config"
	"io"
	"log/slog"
	"sync"
	"time"
)

// OnlineClients 存储的客户端信息
var OnlineClients = sync.Map{}

func DeleteOnlineClient(sessionId string) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("DeleteOnlineClient recover error:", "err_msg", err)
		}
	}()
	cli, ok := OnlineClients.Load(sessionId)
	if !ok || cli == nil {
		slog.Info("OnlineClient sessionId not exist")
		return
	}

	conn, ok := cli.(*SshConn)
	if !ok || conn == nil {
		slog.Error("DeleteOnlineClient type Asset error")
		return
	}

	// 从map 中删除会话
	defer OnlineClients.Delete(sessionId)

	// 关闭 ssh 客户端
	defer func() {
		err := conn.sshClient.Close()
		if err == io.EOF {
			slog.Info("sshClient.Close EOF")
			return
		}
		if err != nil {
			slog.Error("DeleteOnlineClient.Close sshClient error:", "err_msg", err)
		}
	}()

	// 关闭 sftp 客户端
	defer func() {
		err := conn.sftpClient.Close()
		if err == io.EOF {
			slog.Info("sftpClient.Close EOF")
			return
		}
		if err != nil {
			slog.Error("DeleteOnlineClient.Close sftpClient error:", "err_msg", err)
		}
	}()

	// 关闭 ssh 会话
	defer func() {
		err := conn.sshSession.Close()
		if err == io.EOF {
			slog.Info("sshSession.Close EOF")
			return
		}
		if err != nil {
			slog.Error("DeleteOnlineClient.Close sshSession error:", "err_msg", err)
		}
	}()

	// 关闭 websocket
	defer func() {
		err := conn.ws.Close()
		if err != nil {
			slog.Error("DeleteOnlineClient.Close ws error:", "err_msg", err)
		}
	}()

}

// 清理不活跃的会话
func cleanNoActiveSession() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("cleanNoActiveSession error:", "err_msg", err)
		}
	}()
	OnlineClients.Range(func(key, value any) bool {
		// 对键进行类型断言
		if sessionId, ok := key.(string); ok {
			// 对值进行类型断言
			if conn, ok := value.(*SshConn); ok {
				if conn.LastActiveTime.Add(time.Minute).Before(time.Now()) {
					slog.Info("clean not active session:", "sid", sessionId)
					DeleteOnlineClient(sessionId)
				}
			}
		}
		return true
	})
}

func initApp() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("service init error")
		}
	}()
	if config.DefaultConfig.IsInit {
		isStartSshd <- true
	}
	for {
		cleanNoActiveSession()
		time.Sleep(config.DefaultConfig.ClientCheck)
	}
}

func InitSessionClean() {
	go initApp()
}
