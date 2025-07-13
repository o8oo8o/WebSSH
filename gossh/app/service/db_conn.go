package service

import (
	"gossh/app/config"
	"gossh/app/model"
	"gossh/gin"
	"gossh/gorm"
	"gossh/gorm/driver/mysql"
	"gossh/gorm/driver/pgsql"
	_ "gossh/mysql"
	_ "gossh/pgsql"
	"log/slog"
)

type DbConnConf struct {
	DbDsn  string `form:"db_dsn" binding:"required,min=1,max=65535" json:"db_dsn"`
	DbType string `form:"db_type" binding:"required,oneof=sqlite pgsql mysql" json:"db_type"`
}

func DbConnCheck(c *gin.Context) {
	if config.DefaultConfig.IsInit {
		c.JSON(200, gin.H{"code": 1, "msg": "系统已经完成初始化配置"})
		return
	}

	var dbConf DbConnConf
	if err := c.ShouldBind(&dbConf); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	err := DbConnTestCheck(dbConf)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "连接成功"})
}

func DbConnTestCheck(dbConf DbConnConf) error {
	slog.Info("DB link check", "db_type", dbConf.DbType, "db_dsn", dbConf.DbDsn)
	if dbConf.DbType == "pgsql" {
		Db, err := gorm.Open(pgsql.Open(dbConf.DbDsn), &gorm.Config{})
		if err != nil {
			return err
		}
		err = Db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
	}

	if dbConf.DbType == "mysql" {
		Db, err := gorm.Open(mysql.Open(dbConf.DbDsn), &gorm.Config{})
		if err != nil {
			return err
		}
		err = Db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
	}

	if dbConf.DbType == "sqlite" {
		Db, err := model.GetSqliteDb(dbConf.DbDsn)
		if err != nil {
			return err
		}
		err = Db.Exec("select 1=1;").Error
		if err != nil {
			return err
		}
	}
	return nil
}
