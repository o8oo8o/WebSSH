package service

import (
	"encoding/json"
	"fmt"
	"gossh/app/model"
	"gossh/app/utils"
	"gossh/crypto/ssh"
	"gossh/gin"
	"gossh/sftp"
	"gossh/websocket"
	"io"
	"log/slog"
	"net"
	"strconv"
	"time"
)

type SshConn struct {
	*model.SshConf

	//会话ID
	SessionId string `json:"session_id"`

	// 最后活跃时间,心跳
	LastActiveTime time.Time `json:"last_active_time"`

	// 创建连接的时间
	StartTime time.Time `json:"start_time"`

	// 客户端IP
	ClientIP string `json:"client_ip"`

	//ssh客户端
	sshClient *ssh.Client

	//sftp客户端
	sftpClient *sftp.Client

	//ssh会话
	sshSession *ssh.Session

	// websocket 连接
	ws *websocket.Conn
}

// MarshalJSON 重写序列化方法
func (s *SshConn) MarshalJSON() ([]byte, error) {
	type Alias SshConn
	return json.Marshal(&struct {
		Alias
		LastActiveTime string `json:"last_active_time"`
		StartTime      string `json:"start_time"`
		Pwd            string `json:"pwd"`
		CertData       string `json:"cert_data"`
		CertPwd        string `json:"cert_pwd"`
		CreatedAt      uint   ` json:"created_at"`
		UpdatedAt      uint   ` json:"updated_at"`
		DeletedAt      uint   ` json:"deleted_at"`
	}{
		Alias:          (Alias)(*s),
		LastActiveTime: s.LastActiveTime.Format("2006-01-02 15:04:05"),
		StartTime:      s.StartTime.Format("2006-01-02 15:04:05"),
		Pwd:            "",
		CertData:       "",
		CertPwd:        "",
		CreatedAt:      0,
		UpdatedAt:      0,
		DeletedAt:      0,
	})
}

// 连接主机
func (s *SshConn) connect(clientIp string) error {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("ssh connect error:", "err_msg", err)
		}
	}()
	s.ClientIP = clientIp

	config := ssh.ClientConfig{
		User: s.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(s.Pwd),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 30 * time.Second,
	}

	// 证书认证方式
	if s.AuthType == "cert" {
		privateKeyPassword := []byte(s.CertPwd)
		privateKeyBytes := []byte(s.CertData)
		if s.CertPwd != "" {
			// 使用证书空密码登陆
			signer, err := ssh.ParsePrivateKeyWithPassphrase(privateKeyBytes, privateKeyPassword)
			if err != nil {
				slog.Error("ParsePrivateKeyWithPassphrase error:", "err_msg", err.Error())
				return err
			}
			config.Auth = []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			}
		} else {
			// 使用证书有证书密码登陆
			signer, err := ssh.ParsePrivateKey(privateKeyBytes)
			if err != nil {
				slog.Error("ParsePrivateKey error:", "err_msg", err.Error())
				return err
			}
			config.Auth = []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			}
		}
	}

	addr := fmt.Sprintf("%s:%d", s.Address, s.Port)
	if s.NetType == "tcp6" {
		addr = fmt.Sprintf("[%s]:%d", s.Address, s.Port)
	}
	sshClient, err := ssh.Dial(s.NetType, addr, &config)
	if err != nil {
		return err
	}

	s.sshClient = sshClient
	//使用sshClient构建sftpClient
	var sftpClient *sftp.Client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		slog.Error("create sftp sshClient error:", "err_msg", err)
	}
	s.sftpClient = sftpClient

	sshSession, err := s.sshClient.NewSession()
	if err != nil {
		slog.Error("sshClient.NewSession error:", "err_msg", err.Error())
		return err
	}
	s.sshSession = sshSession
	return nil
}

// RunTerminal 运行一个终端
func (s *SshConn) RunTerminal(shell string, stdout, stderr io.Writer, stdin io.Reader, w, h int, ws *websocket.Conn) error {
	defer func() {
		DeleteOnlineClient(s.SessionId)
		if err := recover(); err != nil {
			slog.Error("RunTerminal error:", "err_msg", err)
		}
	}()

	s.ws = ws
	s.sshSession.Stdout = stdout
	s.sshSession.Stderr = stderr
	s.sshSession.Stdin = stdin
	modes := ssh.TerminalModes{}
	if err := s.sshSession.RequestPty(s.PtyType, h, w, modes); err != nil {
		slog.Error("sshSession.RequestPty error:", "err_msg", err.Error())
		_ = websocket.Message.Send(ws, "sshSession.RequestPty error:"+err.Error())
		return err
	}

	err := s.sshSession.Run(shell)
	if err != nil {
		slog.Error("sshSession.Run error:", err.Error())
		_ = websocket.Message.Send(ws, "sshSession.Run error:"+err.Error())
		return err
	}
	return nil
}

// ResizeWindow  调整终端大小
func (s *SshConn) ResizeWindow(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("ResizeWindow recover error:", "err_msg", err)
		}
	}()
	w, err := strconv.Atoi(c.Query("w"))
	if err != nil || (w < 40 || w > 8192) {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  fmt.Sprintf("connect error window width !!!")})
		return
	}
	h, err := strconv.Atoi(c.Query("h"))
	if err != nil || (h < 2 || h > 4096) {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  fmt.Sprintf("connect error window width !!!")})
		return
	}

	sessionId := c.Query("session_id")
	cli, ok := OnlineClients.Load(sessionId)
	if !ok || cli == nil {
		c.JSON(200, gin.H{"code": 1, "msg": "the client is disconnected"})
		return
	}

	conn, ok := cli.(*SshConn)
	if !ok || conn == nil {
		DeleteOnlineClient(sessionId)
		c.JSON(200, gin.H{"code": 1, "msg": "to type SshConn error"})
		return
	}

	err = conn.sshSession.WindowChange(h, w)
	if err != nil {
		DeleteOnlineClient(sessionId)
		slog.Error("sshSession.WindowChange error:", "err_msg", err.Error())
	}
	str := fmt.Sprintf("W:%d;H:%d\n", w, h)
	c.JSON(200, gin.H{"code": 0, "data": str, "msg": "ok"})
	return
}

func ResizeWindow(c *gin.Context) {
	var sshObj = SshConn{}
	sshObj.ResizeWindow(c)
}

func NewSshConn(c *gin.Context) {
	// WebSock 连接 SSH
	websocket.Handler(func(ws *websocket.Conn) {
		sessionId := ws.Request().URL.Query().Get("session_id")
		defer DeleteOnlineClient(sessionId)
		w, err := strconv.Atoi(ws.Request().URL.Query().Get("w"))
		if err != nil || (w < 40 || w > 8192) {
			_ = websocket.Message.Send(ws, "connect error window width !!!")
			DeleteOnlineClient(sessionId)
			return
		}
		h, err := strconv.Atoi(ws.Request().URL.Query().Get("h"))
		if err != nil || (h < 2 || h > 4096) {
			_ = websocket.Message.Send(ws, "connect error window height !!!")
			DeleteOnlineClient(sessionId)
			return
		}

		cli, ok := OnlineClients.Load(sessionId)
		if !ok || cli == nil {
			_ = websocket.Message.Send(ws, "session_id not exists !!!")
			DeleteOnlineClient(sessionId)
			return
		}

		conn, ok := cli.(*SshConn)
		if !ok || conn == nil {
			_ = websocket.Message.Send(ws, "to ssh.Session error !!!")
			DeleteOnlineClient(sessionId)
			return
		}
		err = conn.RunTerminal(conn.Shell, ws, ws, ws, w, h, ws)
		if err != nil {
			_ = websocket.Message.Send(ws, "connect error:"+err.Error())
			DeleteOnlineClient(sessionId)
			return
		}
	}).ServeHTTP(c.Writer, c.Request)
}

func CreateSessionId(c *gin.Context) {
	var conn SshConn
	if err := c.ShouldBind(&conn); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 如果客户提供,使用客户的,并从会话列表删除,防止重复存在
	sessionId := c.Query("session_id")
	//DeleteOnlineClient(sessionId)
	if sessionId == "" {
		sessionId = utils.RandString(15)
	}

	conn.SessionId = sessionId
	conn.LastActiveTime = time.Now()
	conn.StartTime = time.Now()

	err := conn.connect(c.RemoteIP())
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "CreateSessionId error:" + err.Error()})
		return
	}

	OnlineClients.Store(sessionId, &conn)
	c.JSON(200, gin.H{"code": 0, "data": sessionId, "msg": "ok"})
}

func Disconnect(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{
				"code": 1,
				"msg":  "delete connect error",
			})
		}
	}()

	sessionId := c.Query("session_id")
	if sessionId == "" {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "session not exists",
		})
		return
	}
	DeleteOnlineClient(sessionId)
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "delete connect success",
	})
}

func ExecCommand(c *gin.Context) {
	type Param struct {
		SessionId string `form:"session_id" binding:"required,min=10" json:"session_id"`
		Cmd       string `form:"cmd" binding:"required,min=1" json:"cmd"`
	}
	var param Param
	if err := c.ShouldBind(&param); err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}

	cli, ok := OnlineClients.Load(param.SessionId)
	if !ok || cli == nil {
		c.JSON(200, gin.H{"code": 3, "msg": "session not exists"})
		return
	}

	conn, ok := cli.(*SshConn)
	if !ok || conn == nil {
		c.JSON(200, gin.H{"code": 4, "msg": "conn not exists"})
		return
	}

	//创建ssh-session
	session, err := conn.sshClient.NewSession()
	if err != nil {
		c.JSON(200, gin.H{"code": 5, "msg": "create session error"})
		return
	}
	defer func(session *ssh.Session) {
		_ = session.Close()
	}(session)

	//执行命令
	out, err := session.CombinedOutput(param.Cmd)
	if err != nil {
		c.JSON(200, gin.H{"code": 6, "msg": "exec cmd error", "data": string(out)})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": string(out),
	})
}
