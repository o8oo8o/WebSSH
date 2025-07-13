package model

import (
	"gossh/app/config"
	"gossh/app/utils"
	"gossh/gorm"
)

type SshdConf struct {
	ID            uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Name          string   `gorm:"column:name;uniqueIndex;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Host          string   `gorm:"column:host;size:128" form:"host" binding:"required,min=1,max=128" json:"host"`
	Port          uint16   `gorm:"column:port;uniqueIndex;not null" form:"port" binding:"required,gte=1,lte=65535" json:"port"`
	Shell         string   `gorm:"column:shell;not null;size:64;default:''" form:"shell" binding:"min=1,max=128" json:"shell"`
	KeyFile       string   `gorm:"column:key_file;not null;type:text" form:"key_file" json:"key_file"`
	KeySeed       string   `gorm:"column:key_seed;not null;size:4096;default:''" form:"key_seed" binding:"max=4096" json:"key_seed"`
	KeepAlive     uint16   `gorm:"column:keep_alive;not null;default:60" form:"keep_alive" binding:"required,gte=10,lte=360" json:"keep_alive"`
	LoadEnv       string   `gorm:"column:load_env;not null;size:64;default:'Y'" form:"load_env" binding:"required,min=1,max=64,oneof=Y N" json:"load_env"`
	AuthType      string   `gorm:"column:auth_type;not null;size:32;default:'all'" form:"auth_type" binding:"required,min=1,max=32,oneof=all pwd cert" json:"auth_type"`
	ServerVersion string   `gorm:"column:server_version;not null;size:64;default:'SSH-2.0-OpenSSH'" form:"name" binding:"required,min=10,max=63" json:"server_version"`
	CreatedAt     DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt     DateTime `gorm:"column:updated_at" json:"-"`
}

func (c *SshdConf) Create(conf *SshdConf) error {
	return Db.Create(conf).Error
}

func (c *SshdConf) FindByName(name string) (SshdConf, error) {
	var conf SshdConf
	err := Db.Find(&conf, "name = ?", name).Error
	return conf, err
}

func (c *SshdConf) FindByID(id uint) (SshdConf, error) {
	var conf SshdConf
	err := Db.First(&conf, "id = ?", id).Error
	return conf, err
}

func (c *SshdConf) UpdateByName(name string, conf *SshdConf) error {
	return Db.Model(&c).Where("name = ?", name).Select("*").Omit("id", "name").Updates(conf).Error
}

func (c *SshdConf) DeleteByName(name string) error {
	return Db.Unscoped().Delete(&c, "name = ?", name).Error
}

// BeforeCreate Hook
func (c *SshdConf) BeforeCreate(_ *gorm.DB) error {
	var err error
	c.KeyFile, err = utils.EncryptString(c.KeyFile, config.DefaultConfig.AesSecret)
	return err
}

// BeforeUpdate Hook
func (c *SshdConf) BeforeUpdate(_ *gorm.DB) error {
	var err error
	c.KeyFile, err = utils.EncryptString(c.KeyFile, config.DefaultConfig.AesSecret)
	return err
}

// AfterFind Hook
func (c *SshdConf) AfterFind(_ *gorm.DB) error {
	var err error
	c.KeyFile, err = utils.DecryptString(c.KeyFile, config.DefaultConfig.AesSecret)
	return err
}
