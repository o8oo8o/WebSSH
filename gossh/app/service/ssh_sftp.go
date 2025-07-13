package service

import (
	"errors"
	"fmt"
	"gossh/gin"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

func getSshConn(sessionId string) (*SshConn, error) {
	cli, ok := OnlineClients.Load(sessionId)
	if !ok {
		slog.Error("加载ssh连接错误")
		return nil, errors.New("加载ssh连接错误")
	}
	conn, ok := cli.(*SshConn)
	if conn != nil && !ok {
		slog.Error("断言ssh连接错误")
		return nil, errors.New("断言ssh连接错误")
	}
	return conn, nil
}

// SftpList GET sftp 获取指定目录下文件信息
func SftpList(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{"code": 4, "msg": "读取目录错误"})
			return
		}
	}()
	dirPath := c.PostForm("path")

	conn, err := getSshConn(c.PostForm("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}

	files, err := conn.sftpClient.ReadDir(dirPath)
	if err != nil {
		slog.Error("sftp客户端ReadDir错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": "sftp客户端读取目录错误"})
		return
	}

	fileCount := 0
	dirCount := 0
	var fileList []any
	for _, file := range files {
		fileInfo := map[string]any{}
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

	data := map[string]any{
		"files":       fileList,
		"file_count":  fileCount,
		"dir_count":   dirCount,
		"paths":       pathHandler(dirPath),
		"current_dir": dirPath,
	}

	c.JSON(200, gin.H{"code": 0, "data": data, "msg": "ok"})
}

// SftpDownLoad POST sftp 下载文件
func SftpDownLoad(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": "下载错误"})
			return
		}
	}()
	fullPath, err := url.QueryUnescape(c.Query("path"))
	if err != nil {
		slog.Error("获取文件路径参数错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": "获取文件路径参数错误"})
		return
	}
	conn, err := getSshConn(c.Query("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin.H{"code": 3, "msg": err.Error()})
		return
	}

	file, err := conn.sftpClient.Open(fullPath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		slog.Error("sftpClient.Openc错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 4, "msg": "sftp打开文件错误"})
		return
	}

	stat, err := file.Stat()
	if err != nil {
		slog.Error("file.Stat()错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 4, "msg": "读取文件信息错误"})
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename="+stat.Name())
	c.Header("Content-Type", "application/octet-stream")
	//c.Header("Content-Type", "application/x-download")
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))
	_, err = file.WriteTo(c.Writer)
	if err != nil {
		slog.Error("file.WriteTo错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 5, "msg": "下载文件错误"})
		return
	}
	c.Writer.Flush()
}

// SftpUpload PUT sftp 上传文件
func SftpUpload(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{"code": 4, "msg": "上传错误"})
			return
		}
	}()

	dstPath := c.PostForm("path")
	//获取上传的文件组
	form, err := c.MultipartForm()
	if err != nil {
		slog.Error("获取form数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "获取form数据错误"})
		return
	}
	files := form.File["files"]
	// files := c.Request.MultipartForm.File["file"]

	conn, err := getSshConn(c.PostForm("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}

	var ret []string
	for _, file := range files {
		srcFile, err := file.Open()
		if err != nil {
			continue
		}
		fileName := file.Filename
		dstFile, err := conn.sftpClient.Create(path.Join(dstPath, fileName))
		if err != nil {
			continue
		}
		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			continue
		}
		_ = srcFile.Close()
		_ = dstFile.Close()
		ret = append(ret, fileName)
	}
	msg := strconv.Itoa(len(ret)) + " 个文件上传成功"
	c.JSON(200, gin.H{"code": 0, "msg": msg, "data": ret})
}

// SftpDelete DELETE sftp 删除文件或目录
func SftpDelete(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{"code": 4, "msg": "删除错误"})
			return
		}
	}()

	type Body struct {
		SessionId string `form:"session_id" binding:"required,min=1,max=128" json:"session_id"`
		Path      string `form:"path" binding:"required,min=1,max=1024" json:"path"`
	}

	var body Body
	if err := c.ShouldBind(&body); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	conn, err := getSshConn(body.SessionId)
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	err = conn.sftpClient.RemoveAll(body.Path)
	if err != nil {
		slog.Error("sftpClient.Remove错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": "删除文件错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "删除成功"})
}

// SftpCreateDir sftp 创建目录
func SftpCreateDir(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin.H{"code": 4, "msg": "创建目录错误"})
			return
		}
	}()

	type Body struct {
		SessionId string `form:"session_id" binding:"required,min=1,max=128" json:"session_id"`
		Path      string `form:"path" binding:"required,min=1,max=1024" json:"path"`
	}

	var body Body
	if err := c.ShouldBind(&body); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	conn, err := getSshConn(body.SessionId)
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	err = conn.sftpClient.MkdirAll(body.Path)
	if err != nil {
		slog.Error("sftpClient.MkdirAll错误", "err_msg", err.Error())
		c.JSON(200, gin.H{"code": 2, "msg": "创建目录错误"})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "创建目录成功"})
}
