package config

import (
	"flag"
	"gossh/app/utils"
	"gossh/toml"
	"log/slog"
	"os"
	"path"
	"time"
)

type AppConfig struct {
	AppName       string        `json:"app_name"  toml:"app_name"`
	DbType        string        `json:"db_type" toml:"db_type"`
	DbDsn         string        `json:"db_dsn" toml:"db_dsn"`
	IsInit        bool          `json:"is_init" toml:"is_init"`
	JwtSecret     string        `json:"jwt_secret" toml:"jwt_secret"`
	AesSecret     string        `json:"aes_secret" toml:"aes_secret"`
	JwtExpire     time.Duration `json:"jwt_expire" toml:"jwt_expire"`
	StatusRefresh time.Duration `json:"status_refresh" toml:"status_refresh"`
	ClientCheck   time.Duration `json:"client_check" toml:"client_check"`
	SessionSecret string        `json:"session_secret" toml:"session_secret"`
	Address       string        `json:"address" toml:"address"`
	Port          string        `json:"port" toml:"port"`
	CertFile      string        `json:"cert_file" toml:"cert_file"`
	KeyFile       string        `json:"key_file" toml:"key_file"`
}

var DefaultConfig = AppConfig{
	AppName:       "GoWebSHH",
	DbType:        "mysql",
	DbDsn:         "",
	IsInit:        false,
	JwtSecret:     utils.RandString(64),
	AesSecret:     utils.RandString(32),
	SessionSecret: utils.RandString(64),
	JwtExpire:     time.Minute * 120,
	StatusRefresh: time.Second * 3,
	ClientCheck:   time.Second * 15,
	Address:       "",
	Port:          "8899",
	CertFile:      path.Join(WorkDir, "cert.pem"),
	KeyFile:       path.Join(WorkDir, "key.key"),
}

var UserHomeDir, _ = os.UserHomeDir()

var projectName = "GoWebSSH"

var confFileName = "GoWebSSH.toml"

var confPath = string(os.PathSeparator) + "." + projectName + string(os.PathSeparator)

// WorkDir 程序默认工作目录,在用户的home目录下 .GoWebSSH 目录
var WorkDir = path.Join(UserHomeDir, confPath)

var confFileFullPath = path.Join(WorkDir, confFileName)

func init() {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("init error", err)
			os.Exit(255)
		}
	}()

	var dir string
	flag.StringVar(&dir, "WorkDir", "", "自定义工作目录")
	flag.Parse()
	if dir != "" {
		WorkDir = path.Join(dir, confPath)
		confFileFullPath = path.Join(WorkDir, confFileName)
		DefaultConfig.CertFile = path.Join(WorkDir, "cert.pem")
		DefaultConfig.KeyFile = path.Join(WorkDir, "key.key")
	}
	slog.Info("use-config-file", "path", confFileFullPath)

	info, err := os.Stat(WorkDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(WorkDir, os.FileMode(0755))
		if err != nil {
			slog.Error("创建工作目录失败")
			return
		}
	}

	info, err = os.Stat(WorkDir)
	if !info.IsDir() {
		slog.Error("有一个和工作目录同名的文件")
		return
	}

	_, err = os.Stat(confFileFullPath)
	if os.IsNotExist(err) {
		data, err := toml.Marshal(DefaultConfig)
		if err != nil {
			slog.Error("序列化TOML配置文件错误:", "err_msg", err)
			return
		}

		err = os.WriteFile(confFileFullPath, data, os.FileMode(0777))
		if err != nil {
			slog.Error("写入默认配置文件错误:", "err_msg", err.Error())
			return
		}
	}

	data, err := os.ReadFile(confFileFullPath)
	if err != nil {
		slog.Error("读取配置文件错误:", "err_msg", err.Error())
		return
	}

	err = toml.Unmarshal(data, &DefaultConfig)
	if err != nil {
		slog.Error("TOML解析配置文件错误:", "err_msg", err.Error())
		return
	}
	slog.Info("DefaultConfig:", "data", DefaultConfig)

}

func RewriteConfig(conf AppConfig) error {
	data, err := toml.Marshal(conf)
	if err != nil {
		slog.Error("序列化TOML配置文件错误:", "err_msg", err)
		return err
	}
	// 覆盖全局变量
	DefaultConfig = conf
	err = os.Remove(confFileFullPath)
	if err != nil {
		slog.Error("删除旧配置文件错误:", "err_msg", err.Error())
		return err
	}
	err = os.WriteFile(confFileFullPath, data, os.FileMode(0777))
	if err != nil {
		slog.Error("写入默认配置文件错误:", "err_msg", err.Error())
		return err
	}
	return nil
}
