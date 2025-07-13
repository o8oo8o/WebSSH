package model

import (
	"errors"
	"fmt"
	"gossh/app/config"
	"gossh/gorm"
	"gossh/gorm/driver/mysql"
	"gossh/gorm/driver/pgsql"
	"gossh/gorm/logger"
	_ "gossh/mysql"
	_ "gossh/pgsql"
	"gossh/sqlite"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

var Db *gorm.DB

func InitDatabase() {
	if !config.DefaultConfig.IsInit {
		slog.Warn("系统未初始化,跳过DbMigrate")
		return
	}
	err := DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn)
	if err != nil {
		slog.Error("DbMigrate error", "err_msg", err.Error())
	}
}

func GetSqliteDb(dsn string) (*gorm.DB, error) {
	//loadInit()
	dbPath := path.Join(config.WorkDir, dsn)

	// 确保数据库目录存在
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// 尝试打开数据库连接
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// 验证连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

func DbMigrate(dbType, dsn string) error {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("DbMigrate error", "err_msg", err)
		}
	}()
	if dbType == "pgsql" {
		db, err := gorm.Open(pgsql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return err
		}
		err = db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
		Db = db
	}

	if dbType == "mysql" {
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err != nil {
			return err
		}
		err = db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
		Db = db
	}

	if dbType == "sqlite" {
		db, err := GetSqliteDb(dsn)
		if err != nil {
			return err
		}
		Db = db
	}

	if Db == nil {
		return errors.New("请检查数据库链接")
	}

	err := Db.AutoMigrate(
		SshConf{}, WebUser{}, CmdNote{},
		NetFilter{}, PolicyConf{}, LoginAudit{},
		SshdConf{}, SshdUser{}, SshdCert{})
	if err != nil {
		slog.Error("AutoMigrate error:", "err_msg", err.Error())
		return err
	}

	return nil
}
