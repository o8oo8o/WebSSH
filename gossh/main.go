package main

import (
	"bufio"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	succeed = 0
	failure = 1
)

const (
	Emergency LogLevel = iota
	Alert
	Critical
	Error
	Warning
	Notice
	Info
	Debug
)

// 使用go 1.16 新特性
//go:embed webroot
var dir embed.FS

var db *sql.DB

var userHomeDir, _ = os.UserHomeDir()

var projectName = "GoWebSSH"

// 程序默认工作目录,在用户的home目录下 .GoWebSSH 目录
var WorkDir = path.Join(userHomeDir, fmt.Sprintf("/.%s/", projectName))

// 日志级别
var logLevel = Error

// 日志输出到文件
var logOutFile = true

// 日志输出到控制台
var logOutConsole = true

// server默认配置,当配置文件不存在的时候,就使用这个默认配置
var Config = map[string]map[string]string{
	"app": {
		"AppName": "god",
		"Key":     RandString(64),
	},
	"server": {
		"Address":  "0.0.0.0",
		"Port":     "8899",
		"CertFile": path.Join(WorkDir, "cert.pem"),
		"KeyFile":  path.Join(WorkDir, "key.key"),
	},
	"session": {
		"Store":    "memory",
		"Name":     "session_id",
		"Path":     "/",
		"Domain":   "",
		"MaxAge":   "86400",
		"Secure":   "false",
		"HttpOnly": "true",
		"SameSite": "2",
	},
}

//###############################
// 日志功能
//###############################

// brush is a color join function
type brush func(string) string

// newBrush return a fix color Brush
func newBrush(color string) brush {
	return func(text string) string {
		return "\033[" + color + "m" + text + "\033[0m"
	}
}

var colors = []brush{
	newBrush("1;41"), // Emergency          white
	newBrush("1;36"), // Alert              cyan
	newBrush("1;35"), // Critical           magenta
	newBrush("1;31"), // Error              red
	newBrush("1;33"), // Warning            yellow
	newBrush("1;32"), // Notice             green
	newBrush("1;34"), // Informational      blue
	newBrush("1;38"), // Debug              white
}

type LogLevel uint8

type Log struct {
	*log.Logger
	logFile    *os.File
	Name       string
	Level      LogLevel
	OutFile    bool
	OutConsole bool
}

// NewLogger
// name 日志文件名称
// level 日志级别
// outFile 是否把日志输出到文件
// outConsole 是否把日志输出到控制台
func NewLogger(name string, level LogLevel, outFile, outConsole bool) *Log {
	if strings.TrimSpace(name) == "" {
		log.Println("Panic:Log name cannot be empty")
	}

	logFile, err := os.OpenFile(name+".log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Panic:Open log file error")
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Lmicroseconds|log.Lshortfile)

	logObject := &Log{
		logger,
		logFile,
		name,
		level,
		outFile,
		outConsole,
	}
	return logObject
}

func (l *Log) Emergency(v ...interface{}) {
	if l.Level < 0 {
		return
	}
	l.SetPrefix("[M]: ")
	l.write(fmt.Sprint(v...), 0)
}

func (l *Log) Alert(v ...interface{}) {
	if l.Level < 1 {
		return
	}
	l.SetPrefix("[A]: ")
	l.write(fmt.Sprint(v...), 1)
}

func (l *Log) Critical(v ...interface{}) {
	if l.Level < 2 {
		return
	}
	l.SetPrefix("[C]: ")
	l.write(fmt.Sprint(v...), 2)
}

func (l *Log) Error(v ...interface{}) {
	if l.Level < 3 {
		return
	}
	l.SetPrefix("[E]: ")
	l.write(fmt.Sprint(v...), 3)
}

func (l *Log) Warning(v ...interface{}) {
	if l.Level < 4 {
		return
	}
	l.SetPrefix("[W]: ")
	l.write(fmt.Sprint(v...), 4)
}

func (l *Log) Notice(v ...interface{}) {
	if l.Level < 5 {
		return
	}
	l.SetPrefix("[N]: ")
	l.write(fmt.Sprint(v...), 5)
}

func (l *Log) Info(v ...interface{}) {
	if l.Level < 6 {
		return
	}
	l.SetPrefix("[I]: ")
	l.write(fmt.Sprint(v...), 6)
}

func (l *Log) Debug(v ...interface{}) {
	if l.Level < 7 {
		return
	}
	l.SetPrefix("[D]: ")
	l.write(fmt.Sprint(v...), 7)
}

func (l *Log) write(msg string, level int) {
	if l.OutConsole {
		l.SetOutput(os.Stdout)
		_ = l.Output(3, colors[level](msg))
	}

	if l.OutFile {
		l.SetOutput(l.logFile)
		_ = l.Output(3, msg)
	}
}

func (l *Log) SetLogLevel(level LogLevel) {
	l.Level = level
}

// 日志功能
var logger = NewLogger("GoSSH", logLevel, logOutFile, logOutConsole)

/**
// 是否输出到控制台
logger.OutConsole = false

// 是否输出到文件
logger.OutFile = true

// 设置日志级别
logger.SetLogLevel(Error)

logger.Debug("Debug")
logger.Info("Informational")
logger.Notice("Notice")
logger.Warning("Warning")
logger.Error("Error")
logger.Critical("Critical")
logger.Alert("Alert")
logger.Emergency("Emergency")
*/

// 生成指定长度随机字符串
func RandString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	data := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, data[r.Intn(len(data))])
	}
	return string(result)
}

//###############################
// 控制器Base
//###############################

// 默认控制器,所有的控制器都要继承这个控制器
type Controller struct{}

func (controller *Controller) GET(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not  allowed")
}

func (controller *Controller) POST(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) DELETE(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) PUT(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) HEAD(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) PATCH(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) OPTIONS(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

func (controller *Controller) TRACE(c *HttpContext) {
	c.JSON(http.StatusMethodNotAllowed, nil, failure, "request method not allowed")
}

//###############################
// HttpContext 请求信息
//###############################

type HttpContext struct {
	Response   *http.ResponseWriter
	Request    *http.Request
	Path       string
	Method     string
	StatusCode int
	Session    Session
	SessionId  string
}

// 创建Context
func newContext(w *http.ResponseWriter, r *http.Request) *HttpContext {
	context := &HttpContext{
		Response: w,
		Request:  r,
		Path:     r.URL.Path,
		Method:   r.Method,
	}
	logger.Debug("newContext")
	CreateSession(context)
	return context
}

func (c *HttpContext) Query(key string) string {
	_ = c.Request.ParseMultipartForm(32 << 20)
	value := c.Request.Form.Get(key)
	return strings.TrimSpace(value)
}

func (c *HttpContext) Status(code int) {
	c.StatusCode = code
	(*c.Response).WriteHeader(code)
}

func (c *HttpContext) SetHeader(key string, value string) {
	(*c.Response).Header().Set(key, value)
}

// 响应 string 对象
func (c *HttpContext) String(code int, format string, values ...interface{}) {
	c.Status(code)
	c.SetHeader("Content-Type", "text/plain")
	_, _ = (*c.Response).Write([]byte(fmt.Sprintf(format, values...)))
}

// 响应 json 对象
func (c *HttpContext) JSON(statusCode int, data interface{}, code int, msg string) {
	c.SetHeader("Content-Type", "application/json")
	(*c.Response).WriteHeader(statusCode)

	type DataInfo struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}

	byteData, err := json.Marshal(DataInfo{
		Code: code,
		Msg:  msg,
		Data: data,
	})
	if err != nil {
		_, _ = (*c.Response).Write([]byte(`{"code":1,"msg":"json dump error"}`))
		return
	}
	_, _ = (*c.Response).Write(byteData)
}

// 响应二进制文件
func (c *HttpContext) Data(code int, data []byte) {
	c.Status(code)
	_, _ = (*c.Response).Write(data)
}

// 响应HTML文件
func (c *HttpContext) HTML(code int, html string) {
	c.Status(code)
	c.SetHeader("Content-Type", "text/html")
	_, _ = (*c.Response).Write([]byte(html))
}

// 通过反射调用对应的请求方法
func (c *HttpContext) Handler(controller interface{}) {
	// 反射获取控制器信息
	hostController := reflect.ValueOf(controller)
	requestMethod := hostController.MethodByName(c.Method)

	// 判断控制器是否存在对应的方法
	if !requestMethod.IsValid() {
		c.JSON(400, nil, failure, "request method error")
		return
	}

	// 通过反射调用控制器的方法
	requestMethod.Call([]reflect.Value{reflect.ValueOf(c)})
}

//###############################
// 读取配置文件功能
//###############################

type configFile struct {
	fileName string
	comment  []string
}

// 配置片段类型
type Section map[string]string

// 获取片段中的值,(转换成int)
func (s Section) GetInt(key string) (int, error) {
	data := 0
	if val, ok := s[key]; ok {
		data, err := strconv.Atoi(val)
		if err == nil {
			return data, nil
		}
	}
	return data, errors.New("GetInt Error")
}

// 获取片段中的值,(转换成float)
func (s Section) GetFloat(key string) (float64, error) {
	data := 0.0
	if val, ok := s[key]; ok {
		data, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return data, nil
		}
	}
	return data, errors.New("GetFloat Error")
}

// 获取片段中的值,(默认值字符串)
func (s Section) GetString(key string) (string, error) {
	if val, ok := s[key]; ok {
		return val, nil
	}
	return "", errors.New("GetString Error")
}

// 获取片段中的值,(转换成bool)
func (s Section) GetBool(key string) (bool, error) {
	if val, ok := s[key]; ok {
		data, err := strconv.ParseBool(val)
		if err == nil {
			return data, nil
		}
	}
	return false, errors.New("GetBool Error")
}

// 读取配置文件的每一行
func (c configFile) ReadLines() (lines []string, err error) {
	fd, err := os.Open(c.fileName)
	if err != nil {
		return
	}
	defer func() {
		_ = fd.Close()
	}()
	lines = make([]string, 0)
	reader := bufio.NewReader(fd)
	prefix := ""
	var isLongLine bool
	for {
		byteLine, isPrefix, er := reader.ReadLine()
		if er != nil && er != io.EOF {
			return nil, er
		}
		if er == io.EOF {
			break
		}
		line := string(byteLine)
		if isPrefix {
			prefix += line
			continue
		} else {
			isLongLine = true
		}

		line = prefix + line
		if isLongLine {
			prefix = ""
		}
		line = strings.TrimSpace(line)
		// 跳过空白行
		if len(line) == 0 {
			continue
		}
		// 跳过注释行
		var breakLine = false
		for _, v := range c.comment {
			if strings.HasPrefix(line, v) {
				breakLine = true
				break
			}
		}
		if breakLine {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// 获取所有配置
func (c configFile) GetAllConfig() map[string]map[string]string {
	allConfig := make(map[string]map[string]string)
	lines, err := c.ReadLines()
	if err != nil {
		logger.Error(err)
	}
	var section = make(map[string]string, 1)

	for _, line := range lines {
		if line[0] == '[' && line[len(line)-1] == ']' {
			sectionName := line[1 : len(line)-1]
			section = make(map[string]string, 1)
			allConfig[sectionName] = section
		} else {
			configKeyVal := strings.Split(line, "=")
			key := strings.TrimSpace(configKeyVal[0])
			val := strings.TrimSpace(strings.Join(configKeyVal[1:], "="))
			section[key] = val
		}
	}
	return allConfig
}

// 获取某一段配置
func (c configFile) GetSection(section string) (Section, error) {
	if data, ok := c.GetAllConfig()[section]; ok {
		return data, nil
	}
	return map[string]string{}, nil
}

// 加载配置文件
func LoadConfig(filename string, comment []string) (configFile, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("file not exist:", err)
			return configFile{}, err
		}
	}
	return configFile{
		fileName: filename,
		comment:  comment,
	}, nil
}

//###############################
// session 功能
//###############################

// session 配置
type sessionConfig struct {
	sessionStore    string
	sessionName     string
	sessionPath     string
	sessionHttpOnly bool
	sessionSecure   bool
	sessionDomain   string
	sessionMaxAge   int
	sessionSameSite http.SameSite
}

// 创建session配置
func newSessionConfig() sessionConfig {
	var err error
	var sessionName = "session_id"
	var sessionPath = "/"
	var sessionHttpOnly = true
	var sessionSecure = false
	var sessionDomain = ""
	var sessionMaxAge = 3600
	var sessionSameSite = http.SameSiteLaxMode

	if len(Config["session"]["Name"]) > 0 {
		sessionName = Config["session"]["Name"]
	}

	if len(Config["session"]["Path"]) > 0 {
		sessionPath = Config["session"]["Path"]
	}

	if len(Config["session"]["Domain"]) > 0 {
		sessionDomain = Config["session"]["Domain"]
	}

	sessionHttpOnly, err = strconv.ParseBool(Config["session"]["HttpOnly"])
	if err != nil {
		sessionHttpOnly = true
	}

	sessionSecure, err = strconv.ParseBool(Config["session"]["Secure"])
	if err != nil {
		sessionSecure = false
	}

	sessionMaxAge, err = strconv.Atoi(Config["session"]["MaxAge"])
	if err != nil {
		sessionMaxAge = 3600
	}

	sessionSameSiteVal, err := strconv.Atoi(Config["session"]["SameSite"])
	if err != nil {
		sessionSameSite = http.SameSiteLaxMode
	} else {
		sessionSameSite = http.SameSite(sessionSameSiteVal)
	}

	return sessionConfig{
		sessionName:     sessionName,
		sessionPath:     sessionPath,
		sessionHttpOnly: sessionHttpOnly,
		sessionSecure:   sessionSecure,
		sessionDomain:   sessionDomain,
		sessionMaxAge:   sessionMaxAge,
		sessionSameSite: sessionSameSite,
	}
}

// session存储需要实现的接口
type Session interface {
	Set(key string, value interface{})
	Get(key string) interface{}
	Delete(key string)
	Clear()
	Expire(maxAge int)
}

// 内存数据库
var memoryDatabase = newMemoryDatabase()

type memorySession map[string]interface{}

// 创建session存储
func CreateSession(c *HttpContext) {

	config := memoryDatabase.config

	var sessionId string
	cookie, errs := c.Request.Cookie(config.sessionName)
	var sessionData memorySession
	if errs != nil || cookie.Value == "" {

		// 获取app的Key
		appKey := RandString(64)
		if len(Config["App"]["Key"]) > 0 {
			appKey = Config["session"]["Name"]
		}

		sessionId = fmt.Sprintf("%x", sha256.Sum256([]byte( appKey+strconv.FormatInt(time.Now().UnixNano(), 10))))
		sessionData = memorySession{"id": sessionId, "lastAccessTime": time.Now(), "maxAge": config.sessionMaxAge}
		memoryDatabase.data.Store(sessionId, sessionData)

	} else {
		sessionId, _ = url.QueryUnescape(cookie.Value)
		if data, ok := memoryDatabase.data.Load(sessionId); ok {
			if session, ok := data.(memorySession); ok {
				session.Set("lastAccessTime", time.Now())
				session.Set("maxAge", config.sessionMaxAge)
				sessionData = session
			}
		} else {
			logger.Debug("客户端的 session 数据已经在服务端删除了")
			sessionData = memorySession{"id": sessionId, "lastAccessTime": time.Now(), "maxAge": config.sessionMaxAge}
			memoryDatabase.data.Store(sessionId, sessionData)
		}

	}

	c.SessionId = sessionId
	c.Session = &sessionData

	cookie = &http.Cookie{
		Name:     config.sessionName,
		Value:    sessionId,
		Path:     config.sessionPath,
		HttpOnly: config.sessionHttpOnly,
		Secure:   config.sessionSecure,
		Domain:   config.sessionDomain,
		MaxAge:   config.sessionMaxAge,
		SameSite: config.sessionSameSite,
	}
	(*c.Response).Header().Set("Set-Cookie", cookie.String())
	c.Request.AddCookie(cookie)

}

type MemoryDatabase struct {
	data   sync.Map
	config sessionConfig
}

func (m *MemoryDatabase) MemoryDatabaseGC() {
	for {
		time.Sleep(time.Second)
		m.data.Range(func(key, value interface{}) bool {
			if data, ok := value.(memorySession); ok {
				lastAccessTime := data.Get("lastAccessTime").(time.Time).Unix() + int64(data.Get("maxAge").(int))
				if lastAccessTime < time.Now().Unix() {
					logger.Debug("删除过期session")
					m.data.Delete(key)
				}
			}
			return true
		})
	}
}

func newMemoryDatabase() *MemoryDatabase {
	database := &MemoryDatabase{
		data:   sync.Map{},
		config: newSessionConfig(),
	}
	go database.MemoryDatabaseGC()
	return database
}

func (m *memorySession) Set(key string, value interface{}) {
	(*m)[key] = value

}

func (m *memorySession) Get(key string) interface{} {
	return (*m)[key]
}

func (m *memorySession) Delete(key string) {
	delete(*m, key)
}

func (m *memorySession) Clear() {
	*m = memorySession{"lastAccessTime": time.Now(), "maxAge": 0}
}

func (m *memorySession) Expire(maxAge int) {
	(*m)["maxAge"] = maxAge
}

// 存储的客户端信息
type ClientsInfo struct {
	lock sync.RWMutex
	data map[string]*Ssh
}

var clients = ClientsInfo{
	lock: sync.RWMutex{},
	data: make(map[string]*Ssh),
}

type Login struct {
	Id  int
	Pwd string
	Controller
}

// 登陆管理页面
func (l Login) POST(context *HttpContext) {
	pwd := context.Request.FormValue("pwd")

	row := db.QueryRow("select Id,Pwd from config where Id = 1")

	login := new(Login)
	err := row.Scan(&login.Id, &login.Pwd)
	if err != nil {
		logger.Error(err)
		context.JSON(500, nil, failure, "login error")
		return
	}

	if login.Pwd != pwd {
		context.JSON(401, nil, failure, "login password error")
		return
	}

	context.Session.Set("auth", "Y")
	context.JSON(200, nil, succeed, "login success")
}

// (需要登陆认证)修改登陆密码
func (l Login) PATCH(context *HttpContext) {
	auth := context.Session.Get("auth")
	if auth == nil {
		context.JSON(401, nil, failure, "Unauthorized")
		return
	}

	oldPwd := context.Request.FormValue("old_pwd")
	newPwd := context.Request.FormValue("new_pwd")

	row := db.QueryRow("select Id,Pwd from config where Id = 1")

	login := new(Login)
	err := row.Scan(&login.Id, &login.Pwd)
	if err != nil {
		logger.Error(err)
		context.JSON(500, nil, failure, "login error")
		return
	}

	if login.Pwd != oldPwd {
		context.JSON(401, nil, failure, "old password error")
		return
	}

	stmt, _ := db.Prepare(`update config set Pwd=? where id=1`)
	_, err = stmt.Exec(newPwd)
	if err != nil {
		context.JSON(401, nil, failure, "modify password failure")
		return
	}
	context.JSON(200, nil, succeed, "modify password success")
}

func LoginHandler(response http.ResponseWriter, request *http.Request) {
	context := newContext(&response, request)
	context.Handler(Login{})

}

type Status struct {
	Controller
}

// (需要登陆认证)获取已经连接的主机信息
func (s Status) GET(context *HttpContext) {
	auth := context.Session.Get("auth")
	if auth == nil {
		context.JSON(401, nil, failure, "Unauthorized")
		return
	}

	var data []Ssh
	for _, item := range clients.data {
		data = append(data, *item)
	}

	context.JSON(200, data, succeed, "ok")
}

// 更新已经连接的主机信息
func (s Status) POST(context *HttpContext) {
	_ = context.Request.ParseMultipartForm(32 << 20)

	ids := context.Request.PostForm["ids"]
	clients.lock.Lock()
	defer clients.lock.Unlock()
	for _, key := range ids {
		val, ok := clients.data[key]
		if ok {
			val.Timeout = time.Now()
		}
	}
	context.JSON(200, ids, succeed, "ok")
}

// (需要登陆认证)删除已经建立的连接
func (s Status) DELETE(context *HttpContext) {
	auth := context.Session.Get("auth")
	if auth == nil {
		context.JSON(401, nil, failure, "Unauthorized")
		return
	}

	defer func() {
		if err := recover(); err != nil {
			context.JSON(500, nil, failure, "delete connect error")
		}
	}()

	sessionId := context.Query("session_id")
	if sessionId == "" {
		context.JSON(404, nil, failure, "session not exists")
		return
	}

	sshConn, ok := clients.data[sessionId]
	if ok {
		_ = sshConn.sshClient.Close()
		_ = sshConn.sftpClient.Close()
		_ = sshConn.sshSession.Close()
		_ = sshConn.ws.Close()
		clients.lock.Lock()
		delete(clients.data, sessionId)
		clients.lock.Unlock()
	}

	context.JSON(200, nil, succeed, "delete connect success")
}

func StatusHandler(response http.ResponseWriter, request *http.Request) {
	context := newContext(&response, request)
	context.Handler(Status{})
}

type Ssh struct {
	Controller
	IP         string          `json:"ip"`         //IP地址
	Username   string          `json:"username"`   //用户名
	Password   string          `json:"-"`          //密码
	Port       int             `json:"port"`       //端口号
	SessionId  string          `json:"session_id"` //会话ID
	Shell      string          `json:"shell"`
	Timeout    time.Time       `json:"timeout"`
	StartTime  time.Time       `json:"start_time"` // 建立连接的时间
	sshClient  *ssh.Client     //ssh客户端
	sftpClient *sftp.Client    //sftp客户端
	sshSession *ssh.Session    //ssh会话
	ws         *websocket.Conn // websocket 连接

}

// 重写序列化方法
func (s Ssh) MarshalJSON() ([]byte, error) {
	type Alias Ssh
	return json.Marshal(&struct {
		Alias
		Timeout   string `json:"timeout"`
		StartTime string `json:"start_time"`
	}{
		Alias:     (Alias)(s),
		Timeout:   s.Timeout.Format("2006-01-02 15:04:05"),
		StartTime: s.StartTime.Format("2006-01-02 15:04:05"),
	})
}

func NewClient(ip string, username string, password string, port int, shell, sessionId string) *Ssh {
	cli := new(Ssh)
	cli.IP = ip
	cli.Username = username
	cli.Password = password
	cli.Port = port
	cli.Shell = shell
	cli.SessionId = sessionId
	cli.Timeout = time.Now()
	cli.StartTime = time.Now()
	return cli
}

// 连接主机
func (s *Ssh) connect() error {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()

	config := ssh.ClientConfig{
		User: s.Username,
		Auth: []ssh.AuthMethod{ssh.Password(s.Password)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: 30 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", s.IP, s.Port)
	sshClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		return err
	}

	s.sshClient = sshClient
	//使用sshClient构建sftpClient
	var sftpClient *sftp.Client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		logger.Error("create sftp sshClient error:", err)
	}
	s.sftpClient = sftpClient
	return nil
}

// 运行一个终端
func (s *Ssh) RunTerminal(shell string, stdout, stderr io.Writer, stdin io.Reader, w, h int, ws *websocket.Conn) error {
	if s.sshClient == nil {
		if err := s.connect(); err != nil {
			logger.Error(err)
			return err
		}
	}

	session, err := s.sshClient.NewSession()
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	s.sshSession = session
	s.ws = ws

	defer func() {
		clients.lock.Lock()
		delete(clients.data, s.SessionId)
		clients.lock.Unlock()
		_ = session.Close()
	}()

	session.Stdout = stdout
	session.Stderr = stderr
	session.Stdin = stdin

	modes := ssh.TerminalModes{}

	if err := session.RequestPty("xterm-256color", h, w, modes); err != nil {
		return err
	}

	err = session.Run(shell)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

// 调整终端大小
func (s *Ssh) Resize(context *HttpContext) {
	w, err := strconv.Atoi(context.Query("w"))
	if err != nil || (w < 40 || w > 8192) {
		context.JSON(400, nil, failure, "connect error window width !!!")
		return
	}
	h, err := strconv.Atoi(context.Query("h"))
	if err != nil || (h < 2 || h > 4096) {
		context.JSON(400, nil, failure, "connect error window height !!!")
		return
	}

	sessionId := context.Query("session_id")

	clients.lock.RLock()
	cli, ok := clients.data[sessionId]
	clients.lock.RUnlock()

	if ok {
		if cli.sshSession != nil {
			_ = cli.sshSession.WindowChange(h, w)
			str := fmt.Sprintf("W:%d;H:%d\n", w, h)
			context.JSON(200, str, succeed, "ok")
		} else {
			context.JSON(200, nil, succeed, " resize session in the connection ")
		}
		return
	}
	context.JSON(400, nil, failure, "resize error")

}

func SshHandler(response http.ResponseWriter, request *http.Request) {
	// 调整窗口大小
	if request.Method == http.MethodPatch {
		context := newContext(&response, request)
		sshClient := Ssh{}
		sshClient.Resize(context)
		return
	}

	// WebSock 连接 SSH
	websocket.Handler(func(ws *websocket.Conn) {
		sessionId := ws.Request().URL.Query().Get("session_id")
		w, err := strconv.Atoi(ws.Request().URL.Query().Get("w"))
		if err != nil || (w < 40 || w > 8192) {
			_ = websocket.Message.Send(ws, "connect error window width !!!")
			clients.lock.Lock()
			delete(clients.data, sessionId)
			clients.lock.Unlock()
			_ = ws.Close()
			return
		}
		h, err := strconv.Atoi(ws.Request().URL.Query().Get("h"))
		if err != nil || (h < 2 || h > 4096) {
			_ = websocket.Message.Send(ws, "connect error window height !!!")
			clients.lock.Lock()
			delete(clients.data, sessionId)
			clients.lock.Unlock()
			_ = ws.Close()
			return
		}

		clients.lock.RLock()
		cli := clients.data[sessionId]
		clients.lock.RUnlock()

		err = cli.RunTerminal(cli.Shell, ws, ws, ws, w, h, ws)
		if err != nil {
			_ = websocket.Message.Send(ws, "connect error!!!")
			clients.lock.Lock()
			delete(clients.data, sessionId)
			clients.lock.Unlock()
			_ = ws.Close()
			return
		}
	}).ServeHTTP(response, request)

}

type Host struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	User        string `json:"user"`
	Pwd         string `json:"pwd"`
	Port        int    `json:"port"`
	FontSize    int    `json:"font_size"`
	Background  string `json:"background"`
	Foreground  string `json:"foreground"`
	CursorColor string `json:"cursor_color"`
	FontFamily  string `json:"font_family"`
	CursorStyle string `json:"cursor_style"`
	Shell       string `json:"shell"`
	Controller
}

func (host Host) Select() ([]Host, error) {
	rows, err := db.Query(`select Id, Name, Address, User, Pwd, Port,FontSize, Background, Foreground, CursorColor, FontFamily, CursorStyle, Shell from host`)
	var hostList []Host
	if err != nil {
		return hostList, err
	}
	for rows.Next() {
		var h = new(Host)
		err = rows.Scan(&h.Id, &h.Name, &h.Address, &h.User, &h.Pwd, &h.Port, &h.FontSize, &h.Background, &h.Foreground, &h.CursorColor, &h.FontFamily, &h.CursorStyle, &h.Shell)
		if err != nil {
			return hostList, err
		}
		hostList = append(hostList, *h)
	}
	_ = rows.Close()
	return hostList, nil
}

func (host Host) Insert(name, address, user, pwd string, port, fontSize int, background, foreground, cursorColor, fontFamily, cursorStyle, shell string) (int64, error) {
	insertSql := `INSERT INTO host(Name, Address, User, Pwd, Port, FontSize, Background, Foreground, CursorColor, FontFamily, CursorStyle, Shell)  values(?,?,?,?,?,?,?,?,?,?,?,?)`
	stmt, err := db.Prepare(insertSql)
	if err != nil {
		return 0, err
	}
	res, err := stmt.Exec(name, address, user, pwd, port, fontSize, background, foreground, cursorColor, fontFamily, cursorStyle, shell)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId() //返回新增的id号
	if err != nil {
		return 0, err
	}
	return id, err
}

func (host Host) Update(id int, name, address, user, pwd string, port, fontSize int, background, foreground, cursorColor, fontFamily, cursorStyle, shell string) (int64, error) {

	stmt, err := db.Prepare(`update host set Name=?, Address=?, User=?, Pwd=?, Port=?, FontSize=?, Background=?, Foreground=?, CursorColor=?, FontFamily=?, CursorStyle=?, Shell=?  where id=?`)
	if err != nil {
		return 0, err
	}
	res, err := stmt.Exec(name, address, user, pwd, port, fontSize, background, foreground, cursorColor, fontFamily, cursorStyle, shell, id)
	if err != nil {
		return 0, err
	}
	affect, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affect, err
}

func (host Host) Delete(id int) (int64, error) {
	stmt, err := db.Prepare("delete from host where id=?")
	if err != nil {
		return 0, err
	}
	res, err := stmt.Exec(id) //将想删除的id输入进去就可以删除输入的id
	if err != nil {
		return 0, err
	}
	affect, err := res.RowsAffected() //几条数据受影响：返回int64类型数据
	if err != nil {
		return 0, err
	}
	return affect, err
}

func (host Host) Verify(context *HttpContext) (Host, error) {
	name := context.Query("name")
	address := context.Query("address")
	user := context.Query("user")
	pwd := context.Query("pwd")
	port := context.Query("port")
	fontSize := context.Query("font_size")
	background := context.Query("background")
	foreground := context.Query("foreground")
	cursorColor := context.Query("cursor_color")
	fontFamily := context.Query("font_family")
	cursorStyle := context.Query("cursor_style")
	shell := context.Query("shell")

	if len(name) > 60 || len(name) == 0 {
		return Host{}, fmt.Errorf("name input error")
	}

	if len(address) > 60 || len(address) == 0 {
		return Host{}, fmt.Errorf("host input error")
	}

	if len(user) > 60 || len(user) == 0 {
		return Host{}, fmt.Errorf("user input error")
	}

	if len(pwd) > 60 || len(pwd) == 0 {
		return Host{}, fmt.Errorf("pwd input error")
	}
	p, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil {
		return Host{}, fmt.Errorf("port input error")
	}
	if p > 65535 || p < 1 {
		return Host{}, fmt.Errorf("port range input error")
	}

	fontsize, err := strconv.Atoi(strings.TrimSpace(fontSize))
	if err != nil {
		fontsize = 16
	}
	if fontsize > 32 || fontsize < 8 {
		fontsize = 16
	}
	if len(strings.TrimSpace(background)) == 0 {
		background = "#000000"
	}

	if len(strings.TrimSpace(foreground)) == 0 {
		foreground = "#FFFFFF"
	}

	if len(strings.TrimSpace(cursorColor)) == 0 {
		cursorColor = "#FFFFFF"
	}

	if len(strings.TrimSpace(fontFamily)) == 0 {
		fontFamily = "Courier"
	}

	if len(strings.TrimSpace(cursorStyle)) == 0 {
		cursorStyle = "block"
	}

	if len(strings.TrimSpace(shell)) == 0 {
		shell = "bash"
	}

	hostInfo := Host{
		Name:        name,
		Address:     address,
		User:        user,
		Pwd:         pwd,
		Port:        p,
		FontSize:    fontsize,
		Background:  background,
		Foreground:  foreground,
		CursorColor: cursorColor,
		FontFamily:  fontFamily,
		CursorStyle: cursorStyle,
		Shell:       shell,
	}
	return hostInfo, nil
}

func (host Host) GET(context *HttpContext) {
	allHost, err := host.Select()
	if err != nil {
		context.JSON(500, nil, failure, err.Error())
	} else {
		context.JSON(200, allHost, succeed, "ok")
	}
}

func (host Host) POST(context *HttpContext) {
	h, err := host.Verify(context)
	if err != nil {
		context.JSON(400, nil, failure, err.Error())
		return
	}
	_, err = host.Insert(h.Name, h.Address, h.User, h.Pwd, h.Port, h.FontSize, h.Background, h.Foreground, h.CursorColor, h.FontFamily, h.CursorStyle, h.Shell)
	if err != nil {
		context.JSON(500, nil, failure, err.Error())
		return
	}
	h.GET(context)
}

func (host Host) PUT(context *HttpContext) {
	host, err := host.Verify(context)
	if err != nil {
		context.JSON(400, nil, failure, err.Error())
		return
	}
	id, err := strconv.Atoi(strings.TrimSpace(context.Request.PostFormValue("id")))
	if err != nil {
		context.JSON(400, nil, failure, err.Error())
		return
	}

	_, err = host.Update(id, host.Name, host.Address, host.User, host.Pwd, host.Port, host.FontSize, host.Background, host.Foreground, host.CursorColor, host.FontFamily, host.CursorStyle, host.Shell)
	if err != nil {
		context.JSON(500, nil, failure, err.Error())
		return
	}
	host.GET(context)
}

func (host Host) DELETE(context *HttpContext) {
	id, err := strconv.Atoi(strings.TrimSpace(context.Request.PostFormValue("id")))
	if err != nil {
		context.JSON(400, nil, failure, err.Error())
		return
	}
	_, err = host.Delete(id)
	if err != nil {
		context.JSON(500, nil, failure, err.Error())
		return
	}
	host.GET(context)
}

func (host Host) PATCH(context *HttpContext) {
	h, err := host.Verify(context)
	if err != nil {
		context.JSON(400, nil, failure, err.Error())
		return
	}
	sessionId := RandString(15)
	client := NewClient(h.Address, h.User, h.Pwd, h.Port, h.Shell, sessionId)

	clients.lock.Lock()
	clients.data[sessionId] = client
	clients.lock.Unlock()
	context.JSON(200, sessionId, succeed, "ok")
}

func HostHandler(response http.ResponseWriter, request *http.Request) {
	context := newContext(&response, request)
	context.Handler(Host{})
}

// 文件上传下载
type File struct {
	Controller
}

// sftp 下载文件
func (file File) POST(context *HttpContext) {
	sessionId := context.Query("session_id")
	fullPath := context.Query("path")
	clients.lock.RLock()
	defer clients.lock.RUnlock()
	cli, ok := clients.data[sessionId]
	if ok {
		file, _ := cli.sftpClient.Open(fullPath)
		defer func() {
			_ = file.Close()
		}()
		_, _ = io.Copy(*context.Response, file)
	}
}

// sftp 获取指定目录下文件信息
func (file File) GET(context *HttpContext) {
	dirPath := context.Query("path")
	sessionId := context.Query("session_id")
	clients.lock.RLock()
	defer clients.lock.RUnlock()
	cli, ok := clients.data[sessionId]
	if !ok {
		context.JSON(400, nil, failure, "list Folder error")
		return
	}
	files, err := cli.sftpClient.ReadDir(dirPath)
	if err != nil {
		context.JSON(400, nil, failure, "sftpClient error")
		return
	}

	fileCount := 0
	dirCount := 0
	var fileList []interface{}
	for _, file := range files {
		fileInfo := map[string]interface{}{}
		fileInfo["path"] = path.Join(dirPath, file.Name())
		fileInfo["name"] = file.Name()
		fileInfo["mode"] = file.Mode().String()
		fileInfo["size"] = file.Size()
		fileInfo["mod_time"] = file.ModTime().Format("2006-01-02 15:04:05")
		if file.IsDir() {
			fileInfo["type"] = "d"
			dirCount += 1
		} else {
			fileInfo["type"] = "f"
			fileCount += 1
		}
		fileList = append(fileList, fileInfo)
	}

	// 内部方法,处理路径信息
	pathHandler := func(dirPath string) (paths []map[string]string) {
		tmp := strings.Split(dirPath, "/")

		var dirs []string
		if strings.HasPrefix(dirPath, "/") {
			dirs = append(dirs, "/")
		}

		for _, item := range tmp {
			name := strings.TrimSpace(item)
			if len(name) > 0 {
				dirs = append(dirs, name)
			}
		}

		for i, item := range dirs {
			fullPath := path.Join(dirs[:i+1]...)
			pathInfo := map[string]string{}
			pathInfo["name"] = item
			pathInfo["dir"] = fullPath
			paths = append(paths, pathInfo)
		}
		return paths
	}

	data := map[string]interface{}{
		"files":       fileList,
		"file_count":  fileCount,
		"dir_count":   dirCount,
		"paths":       pathHandler(dirPath),
		"current_dir": dirPath,
	}

	context.JSON(200, data, succeed, "ok")
}

// sftp 上传文件
func (file File) PUT(context *HttpContext) {
	sessionId := context.Query("session_id")
	dstPath := context.Query("path")
	//获取上传的文件组
	files := context.Request.MultipartForm.File["file"]

	clients.lock.RLock()
	defer clients.lock.RUnlock()
	for _, file := range files {
		cli, ok := clients.data[sessionId]
		if ok {
			srcFile, _ := file.Open()
			dstFile, _ := cli.sftpClient.Create(path.Join(dstPath, file.Filename))
			_, _ = io.Copy(dstFile, srcFile)
			_ = srcFile.Close()
			_ = dstFile.Close()
		}
	}
	msg := strconv.Itoa(len(files)) + "文件上传成功"
	context.JSON(200, nil, succeed, msg)
}

func FileHandler(response http.ResponseWriter, request *http.Request) {
	context := newContext(&response, request)
	context.Handler(File{})
}

// 访问前端编译完成的静态文件
func static(w http.ResponseWriter, r *http.Request) {

	indexHtmlData, err := dir.ReadFile("webroot/index.html")
	if err != nil {
		_, _ = w.Write([]byte("read index.html file error"))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// 请求根路径
	if strings.TrimSpace(r.URL.Path) == "/" {
		_, _ = w.Write(indexHtmlData)
		return
	}

	// 请求根路径下文件
	fileFullPath := fmt.Sprintf("webroot%s", strings.TrimSpace(r.URL.Path))

	mimeType := mime.TypeByExtension(filepath.Ext(fileFullPath))
	w.Header().Set("Content-Type", mimeType)

	data, err := dir.ReadFile(fileFullPath)
	if err != nil {
		// 如果出错直接返回index.html内容,避免出现404问题
		_, _ = w.Write(indexHtmlData)
		return
	}
	_, _ = w.Write(data)
}

/**
清理已经断开的连接
*/
func ConnectGC() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()

	for {
		time.Sleep(time.Second)
		duration, _ := time.ParseDuration("-1m")
		longAgo := time.Now().Add(duration)
		for key, item := range clients.data {
			if item.Timeout.Before(longAgo) {
				_ = item.sshClient.Close()
				_ = item.sftpClient.Close()
				_ = item.sshSession.Close()
				_ = item.ws.Close()
				clients.lock.Lock()
				delete(clients.data, key)
				clients.lock.Unlock()
			}
		}
	}
}

// loadConfigFileData 加载配置文件数据
func loadConfigFileData() {
	configFile, err := LoadConfig(path.Join(WorkDir, "GoWebSSH.cnf"), []string{"#", ";"})
	if err == nil {
		logger.Debug("读取配置文件成功,使用系统配置文件")
		Config = configFile.GetAllConfig()
	}
}

func Main() {
	var err error

	// 处理前端静态文件
	http.HandleFunc("/", static)

	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/host", HostHandler)
	http.HandleFunc("/api/ssh", SshHandler)
	http.HandleFunc("/api/file", FileHandler)
	http.HandleFunc("/api/status", StatusHandler)

	address := fmt.Sprintf("%s:%s", Config["server"]["Address"], Config["server"]["Port"])

	certFile := Config["server"]["CertFile"]
	keyFile := Config["server"]["KeyFile"]

	_, certErr := os.Open(certFile)
	_, keyErr := os.Open(keyFile)

	// 如果证书和私钥文件存在,就使用https协议,否则使用http协议
	if certErr == nil && keyErr == nil {
		logger.Debug("https://{IP}:" + Config["server"]["Port"])
		err = http.ListenAndServeTLS(address, certFile, keyFile, nil)
		if err != nil {
			logger.Error("ListenAndServeTLSError:", err.Error())
			os.Exit(1)
			return
		}
	} else {
		logger.Debug("http://{IP}:" + Config["server"]["Port"])
		err = http.ListenAndServe(address, nil)
		if err != nil {
			logger.Error("ListenAndServeError:", err.Error())
			os.Exit(1)
			return
		}
	}
}

func init() {
	var err error
	fileInfo, err := os.Stat(WorkDir)

	if os.IsNotExist(err) {
		err = os.Mkdir(WorkDir, fs.ModePerm)
		if err != nil {
			logger.Error(fmt.Sprintf("创建目录:%s 失败,%s\n", WorkDir, err))
			os.Exit(1)
			return
		}

	} else {
		if !fileInfo.IsDir() {
			logger.Error(fmt.Sprintf("请删除:%s文件\n", WorkDir))
			os.Exit(1)
			return
		}
	}

	configFilePath := path.Join(WorkDir, projectName+".cnf")
	_, err = os.Stat(configFilePath)
	if os.IsNotExist(err) {
		file, err := os.Create(configFilePath)
		if err != nil {
			logger.Error(fmt.Sprintf("创建默认配置文件:%s 失败,%s\n", configFilePath, err))
			os.Exit(1)
			return
		}
		defer func() {
			_ = file.Close()
		}()

		configContent := `
[app]
AppName=god
Key=` + RandString(64) + `

[server]
Address=0.0.0.0
Port=8899
CertFile=` + path.Join(WorkDir, "cert.pem") + `
KeyFile=` + path.Join(WorkDir, "key.key") + `

[session]
Store=memory
Name=session_id
Path=/
Domain=
MaxAge=86400
Secure=false
HttpOnly=true
SameSite=2
`
		_, err = file.WriteString(configContent)
		if err != nil {
			logger.Error(fmt.Sprintf("写入配置文件:%s 失败,%s\n", configFilePath, err))
			os.Exit(1)
			return
		}
		_ = file.Sync()
	}

	db, err = sql.Open("sqlite3", path.Join(WorkDir, projectName+".db"))
	if err != nil {
		logger.Error(fmt.Sprintf("创建数据库文件:%s失败\n", path.Join(WorkDir, projectName+".db")))
		os.Exit(1)
		return
	}

	createHostTable := `
CREATE TABLE IF NOT EXISTS 'host'
(
    'Id'          INTEGER PRIMARY KEY AUTOINCREMENT,
    'Name'        VARCHAR(32) NOT NULL UNIQUE,
    'Address'     VARCHAR(64) NULL,
    'User'        VARCHAR(64) NULL,
    'Pwd'         VARCHAR(64) NULL,
    'Port'        INT         NOT NULL DEFAULT 22,
    'FontSize'    INT         NOT NULL DEFAULT 14,
    'Background'  VARCHAR(32) NOT NULL DEFAULT '#000000',
    'Foreground'  VARCHAR(32) NOT NULL DEFAULT '#FFFFFF',
    'CursorColor' VARCHAR(32) NOT NULL DEFAULT '#FFFFFF',
    'FontFamily'  VARCHAR(32) NOT NULL DEFAULT 'Courier',
    'CursorStyle' VARCHAR(32) NOT NULL DEFAULT 'block',
    'Shell'       VARCHAR(32) NOT NULL DEFAULT 'bash'
);
`
	_, err = db.Exec(createHostTable)
	if err != nil {
		logger.Error(err)
	}

	createConfigTable := `
CREATE TABLE IF NOT EXISTS 'config'
(
    'Id'          INTEGER PRIMARY KEY AUTOINCREMENT,
    'Pwd'         VARCHAR(64) NOT NULL DEFAULT 'admin'
);
`
	_, err = db.Exec(createConfigTable)
	if err != nil {
		logger.Error(err)
	}

	insertSql := `INSERT INTO config(Id,Pwd)  values(?,?)`
	stmt, _ := db.Prepare(insertSql)
	_, err = stmt.Exec(1, "admin")

	loadConfigFileData()
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			logger.Emergency(err)
		}
	}()
	go ConnectGC()
	Main()
}
