package service

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"gossh/app/model"
	"gossh/crypto/ssh"
	"gossh/pty"
	"gossh/sftp"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Server struct {
	cli    *model.SshdConf
	config *ssh.ServerConfig
}

func NewServer(c *model.SshdConf) (*Server, error) {
	s := &Server{cli: c}
	sc, err := s.computeSSHConfig()
	if err != nil {
		return nil, err
	}
	s.config = sc
	return s, nil
}

func (s *Server) Start() error {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.cli.Host, s.cli.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on" + fmt.Sprintf("%s:%d", s.cli.Host, s.cli.Port))
	}
	slog.Info("Listening on", "host", s.cli.Host, "port", s.cli.Port)
	for {
		tcpConn, err := listen.Accept()
		if err != nil {
			slog.Error("Failed to accept incoming connection", "err", err)
			continue
		}
		go s.handleConn(tcpConn)
	}
}

func (s *Server) handleConn(tcpConn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("recovered from panic in handleConn", "err", err)
		}
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, s.config)
	if err != nil {
		if err != io.EOF {
			slog.Info("Failed to handshake", "err", err)
		}
		return
	}
	slog.Info("New SSH connection from", "address", sshConn.RemoteAddr(), "client_version", sshConn.ClientVersion())
	go ssh.DiscardRequests(reqs)
	go s.handleChannels(chans)
}

func (s *Server) handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go s.handleChannel(newChannel)
	}
}

func (s *Server) handleChannel(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		_ = newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	slog.Info("Channel request", "type", newChannel.ChannelType())
	if d := newChannel.ExtraData(); len(d) > 0 {
		slog.Info("Channel data", "data", d)
	}

	connection, requests, err := newChannel.Accept()
	if err != nil {
		slog.Error("Could not accept channel failed", "err", err)
		return
	}
	log.Println("Channel accepted")
	go s.handleRequests(connection, requests)
}

func (s *Server) handleRequests(connection ssh.Channel, requests <-chan *ssh.Request) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("handleRequests recovered from panic", "err", err)
		}
	}()

	// start keep alive loop
	if ka := s.cli.KeepAlive; ka > 0 {
		ticking := make(chan bool, 1)
		interval := time.Duration(ka) * time.Second
		go s.keepAlive(connection, interval, ticking)
		defer close(ticking)
	}
	// prepare to handle client requests
	env := os.Environ()
	resizes := make(chan []byte, 10)
	defer close(resizes)

	for req := range requests {
		switch req.Type {
		case "subsystem":
			if string(req.Payload[4:]) == "sftp" {
				req.Reply(true, nil)
				server, err := sftp.NewServer(connection, sftp.WithDebug(os.Stderr))
				if err != nil {
					slog.Error("create SFTP server error", "err", err)
					connection.Close()
					return
				}
				err = server.Serve()
				if err == io.EOF {
					server.Close()
					connection.Close()
					return
				} else if err != nil {
					slog.Error("SFTP server error", "err", err)
					connection.Close()
					return
				} else {
					connection.Close()
					return
				}
			}
		case "pty-req":
			termLen := req.Payload[3]
			resizes <- req.Payload[termLen+4:]
			req.Reply(true, nil)
		case "window-change":
			resizes <- req.Payload
		case "env":
			e := struct{ Name, Value string }{}
			_ = ssh.Unmarshal(req.Payload, &e)
			kv := e.Name + "=" + e.Value
			slog.Info("env", "data", kv)
			if s.cli.LoadEnv == "Y" {
				env = appendEnv(env, kv)
			}
		case "shell":
			if len(req.Payload) > 0 {
				slog.Info("shell command ignored", "payload", req.Payload)
			}
			slog.Info("sshd attachShell")
			err := s.attachShell(connection, env, resizes)
			if err != nil {
				slog.Error("exec shell", "err", err)
			}
			req.Reply(err == nil, []byte{})
		case "exec":
			if req.WantReply {
				req.Reply(true, nil)
			}
			command := string(req.Payload[4:])
			slog.Info("exec command string", "cmd", command)
			ret := ExecuteForChannel(command, connection)
			slog.Info("exec return", "code", ret)
			connection.Close()
			return
		default:
			slog.Info("unknown request", "type", req.Type, "want_reply", req.WantReply, "payload", req.Payload)
		}
	}
}

func (s *Server) keepAlive(connection ssh.Channel, interval time.Duration, ticking <-chan bool) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("keepAlive recovered from panic", "err", err)
		}
	}()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_, err := connection.SendRequest("ping", false, nil)
			if err != nil {
				slog.Error("failed to send keep alive ping", "err", err)
			}
			slog.Info("sent keep alive ping")
		case <-ticking:
			return
		}
	}
}

func (s *Server) attachShell(connection ssh.Channel, env []string, resizes <-chan []byte) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("recovered from panic in handleRequests: %s", err)
		}
	}()

	shell := exec.Command(s.cli.Shell)
	dir, err := os.UserHomeDir()
	if err == nil {
		shell.Dir = dir
	}

	shell.Env = env
	slog.Info("Session env", "data", env)

	closeFunc := func() {
		err := connection.Close()
		if err != nil {
			slog.Error("close connect failed", "err_msg", err)
			return
		}
		slog.Info("Session closed")
	}

	shellPty, err := pty.Start(shell)
	if err != nil {
		closeFunc()
		return fmt.Errorf("could not start pty (%s)\n", err)
	}
	//dequeue resizes
	go func() {
		for payload := range resizes {
			w, h := parseDims(payload)
			SetWindowSize(shellPty, w, h)
		}
	}()
	//pipe session to shell and visa-versa
	var once sync.Once
	go func() {
		_, _ = io.Copy(connection, shellPty)
		once.Do(closeFunc)
	}()
	go func() {
		_, _ = io.Copy(shellPty, connection)
		once.Do(closeFunc)
	}()
	slog.Info("shell attached")
	go func() {
		if shell.Process != nil {
			if ps, err := shell.Process.Wait(); err != nil && ps != nil {
				slog.Error("Failed to exit shell", "err", err)
			}
			// shellPty.Close()
		}
		slog.Info("Shell terminated and Session closed")
	}()
	return nil
}

func (s *Server) loadCertAuthFile() (map[string]string, error) {
	var tmp model.SshdCert
	authorizedKeys, err := tmp.GetAuthorizedKeys()
	if err != nil {
		slog.Error("GetAuthorizedKeys failed", "err_msg", err.Error())
		return nil, err
	}

	keys, err := parseKeys([]byte(authorizedKeys))
	if err != nil {
		slog.Error("parseKeys failed", "err_msg", err.Error())
		return nil, err
	}
	return keys, nil
}

func (s *Server) computeSSHConfig() (*ssh.ServerConfig, error) {
	sc := &ssh.ServerConfig{}
	if s.cli.Shell == "" {
		if runtime.GOOS == "windows" {
			s.cli.Shell = "powershell"
		} else {
			s.cli.Shell = "sh"
		}
	}
	p, err := exec.LookPath(s.cli.Shell)
	if err != nil {
		return nil, fmt.Errorf("find shell failed: %s\n", s.cli.Shell)
	}
	s.cli.Shell = p
	slog.Info("Session shell", "shell", s.cli.Shell)

	//user provided key (can generate with 'ssh-keygen -t rsa')
	pri, err := ssh.ParsePrivateKey([]byte(s.cli.KeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key\n")
	}
	sc.AddHostKey(pri)
	slog.Info("RSA key", "fingerprint", fingerprint(pri.PublicKey()))

	// 注册账号密码认证逻辑(回调函数)
	sc.PasswordCallback = func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		var sshdUsers model.SshdUser
		u, err := sshdUsers.FindByNameAndPwd(conn.User(), string(pass))
		if err != nil {
			slog.Error("Authentication failed", "username", conn.User(), "err_msg", err)
			return nil, err
		}

		if conn.User() == u.Name && string(pass) == u.Pwd {
			slog.Info("Authentication failed with password", "username", conn.User())
			return nil, nil
		}
		slog.Info("Authentication failed", "username", conn.User(), "password", u.Pwd)
		return nil, fmt.Errorf("auth denied\n")
	}

	// 注册证书认证逻辑(回调函数)
	sc.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		ks, err := s.loadCertAuthFile()
		if err != nil {
			return nil, err
		}
		slog.Info("Authentication enabled public keys", "size", len(ks))
		return nil, s.matchKeys(key, ks)
	}

	// 注册BannerCallback(回调函数)
	sc.BannerCallback = func(conn ssh.ConnMetadata) string {
		// 写入自定义的横幅信息到客户端
		return fmt.Sprintf("Welcome to the Go_SSH_Server\n=>user:(%s)\n=>remote addr:(%s)\n=>local  addr:(%s)\n=>client  ver:(%s)\n=>server  ver:(%s)\n\n",
			conn.User(), conn.RemoteAddr().String(), conn.LocalAddr(),
			conn.ClientVersion(), conn.ServerVersion())
	}
	sc.ServerVersion = "SSH-2.0-OpenSSH"
	cf, err := s.cli.FindByID(1)
	if err == nil {
		sc.ServerVersion = cf.ServerVersion
	}
	return sc, nil
}

func (s *Server) matchKeys(key ssh.PublicKey, keys map[string]string) error {
	if cmt, exists := keys[string(key.Marshal())]; exists {
		slog.Info("User authenticated with public key", "username", cmt, "public_key", fingerprint(key))
		return nil
	}
	slog.Error("User authentication failed with public key", "err", fingerprint(key))
	return fmt.Errorf("denied\n")
}

func appendEnv(env []string, kv string) []string {
	p := strings.SplitN(kv, "=", 2)
	k := p[0] + "="
	for i, e := range env {
		if strings.HasPrefix(e, k) {
			env[i] = kv
			return env
		}
	}
	return append(env, kv)
}

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// SetWindowSize sets the size of the given pty.
func SetWindowSize(t pty.Pty, w, h uint32) {
	ws := &pty.Winsize{Rows: uint16(h), Cols: uint16(w)}
	_ = pty.Setsize(t, ws)
}

func generateKey(seed string) ([]byte, error) {
	var r io.Reader
	if seed == "" {
		r = rand.Reader
	} else {
		r = newDetermRand([]byte(seed))
	}
	privateKey, err := rsa.GenerateKey(r, 2048)
	if err != nil {
		return nil, err
	}
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}
	b := x509.MarshalPKCS1PrivateKey(privateKey)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: b}), nil
}

func parseKeys(b []byte) (map[string]string, error) {
	lines := bytes.Split(b, []byte("\n"))
	//parse each line
	keys := map[string]string{}
	for _, l := range lines {
		if key, cmt, _, _, err := ssh.ParseAuthorizedKey(l); err == nil {
			keys[string(key.Marshal())] = cmt
		}
	}
	//ensure we got something
	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys found\n")
	}
	return keys, nil
}

func fingerprint(k ssh.PublicKey) string {
	bytesData := sha256.Sum256(k.Marshal())
	b64 := base64.StdEncoding.EncodeToString(bytesData[:])
	if strings.HasSuffix(b64, "=") {
		b64 = strings.TrimSuffix(b64, "=") + "."
	}
	return "SHA256:" + b64
}

func newDetermRand(seed []byte) io.Reader {
	const randSize = 2048
	var out []byte
	//strengthen seed
	var next = seed
	for i := 0; i < randSize; i++ {
		next, out = hash(next)
	}
	return &determRand{
		next: next,
		out:  out,
	}
}

type determRand struct {
	next, out []byte
}

func (d *determRand) Read(b []byte) (int, error) {
	l := len(b)
	//HACK: combat https://golang.org/src/crypto/rsa/rsa.go#L257
	if l == 1 {
		return 1, nil
	}
	n := 0
	for n < l {
		next, out := hash(d.next)
		n += copy(b[n:], out)
		d.next = next
	}
	return n, nil
}

func hash(input []byte) (next []byte, output []byte) {
	nextout := sha512.Sum512(input)
	return nextout[:sha512.Size/2], nextout[sha512.Size/2:]
}

// ExecuteForChannel Execute a process for the channel.
func ExecuteForChannel(shellCmd string, ch ssh.Channel) uint32 {
	var err error

	// Windows 下特殊处理命令
	var proc *exec.Cmd
	if runtime.GOOS == "windows" {
		// 使用 Command 创建命令，并正确处理参数
		proc = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; $OutputEncoding = [System.Text.Encoding]::UTF8; chcp 65001 | Out-Null; "+shellCmd)
	} else {
		proc = exec.Command("sh", "-c", shellCmd)
	}

	proc.Env = append(os.Environ(), "LANG=zh_CN.UTF-8", "LC_ALL=zh_CN.UTF-8")

	exe, err := os.Executable()
	if err != nil {
		slog.Error("get os.Executable error", "err", err)
	}
	slog.Info("os.Executable() exe:", "path", exe)

	dir := filepath.Dir(exe)
	slog.Info("exec dir info:", "path", dir)

	proc.Dir = dir
	if userInfo, err := user.Current(); err == nil {
		proc.Dir = userInfo.HomeDir
	}

	stdin, err := proc.StdinPipe()
	if err != nil {
		slog.Error("create pipe failed", "err", err)
		return 1
	}

	go func() {
		defer stdin.Close()
		_, _ = io.Copy(stdin, ch)
	}()
	proc.Stdout = ch
	proc.Stderr = ch.Stderr()

	// 执行命令
	err = proc.Start()
	if err != nil {
		slog.Error("start command failed", "err", err)
		_, _ = ch.Write([]byte(fmt.Sprintf("start command failed: %v\n", err)))
		return 1
	}

	// 等待命令完成
	err = proc.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			slog.Error("wait command failed", "exit code", exitErr.ExitCode())
			return uint32(exitErr.ExitCode())
		}
		slog.Error("exec command failed", "err", err)
		_, _ = ch.Write([]byte(fmt.Sprintf("exec command failed: %v\n", err)))
		return 1
	}
	slog.Info("exec command success")
	return 0
}

func startSshServer() {
	status := <-isStartSshd
	if status {
		slog.Info("start sshd server")
	}

	var sshdConf model.SshdConf
	conf, err := sshdConf.FindByID(1)
	if err != nil {
		slog.Error("find sshd server config failed", "err_msg", err)
		return
	}
	slog.Info("find sshd server config", "conf", conf)
	s, err := NewServer(&conf)
	if err != nil {
		slog.Error("NewServer failed", "err_msg", err)
		return
	}
	slog.Info("start sshd server")
	err = s.Start()
	if err != nil {
		slog.Error("sshd server start failed", "err_msg", err)
		return
	}
}

var isStartSshd = make(chan bool)

func InitSshServer() {
	go startSshServer()
	slog.Info("exec sshd init func")
}
