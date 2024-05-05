package model

import (
	"errors"
	"gossh/app/config"
	"gossh/gorm"
	"gossh/gorm/driver/mysql"
	"gossh/gorm/driver/pgsql"
	_ "gossh/mysql"
	_ "gossh/pgsql"
	"log/slog"
)

var Db *gorm.DB

func init() {
	if !config.DefaultConfig.IsInit {
		slog.Warn("系统未初始化,跳过DbMigrate")
		return
	}
	err := DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn)
	if err != nil {
		slog.Error("DbMigrate error", "err_msg", err.Error())
	}
}

func DbMigrate(dbType, dsn string) error {
	if dbType == "pgsql" {
		db, err := gorm.Open(pgsql.Open(dsn), &gorm.Config{})
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
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return err
		}
		err = db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
		Db = db
	}

	if Db == nil {
		return errors.New("请检查数据库链接")
	}

	err := Db.AutoMigrate(SshConf{}, SshUser{}, CmdNote{}, NetFilter{}, PolicyConf{}, LoginAudit{})
	if err != nil {
		slog.Error("AutoMigrate error:", "err_msg", err.Error())
		return err
	}

	return nil
}
