package model

import (
	"errors"
	"gossh/app/config"
	"gossh/app/utils"
	"gossh/gorm"
)

type SshConf struct {
	ID          uint   `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Uid         uint   `gorm:"column:uid;not null;default:0" form:"uid" json:"uid"`
	Name        string `gorm:"column:name;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Address     string `gorm:"column:address;size:128" form:"address" binding:"required,min=1,max=128" json:"address"`
	User        string `gorm:"column:user;size:128" form:"user" binding:"required,min=1,max=128" json:"user"`
	Pwd         string `gorm:"column:pwd;not null;size:4096;default:''" form:"pwd" binding:"max=4096" json:"pwd"`
	AuthType    string `gorm:"column:auth_type;not null;size:32;default:'pwd'" form:"auth_type" binding:"required,min=1,max=32,oneof=pwd cert" json:"auth_type"`
	NetType     string `gorm:"column:net_type;not null;size:32;default:'tcp4'" form:"net_type" binding:"required,min=1,max=32,oneof=tcp4 tcp6" json:"net_type"`
	CertData    string `gorm:"column:cert_data;type:text" form:"cert_data" json:"cert_data"`
	CertPwd     string `gorm:"column:cert_pwd;not null;size:128;default:''" form:"cert_pwd" binding:"max=128" json:"cert_pwd"`
	Port        uint16 `gorm:"column:port;not null;default:22" form:"port" binding:"required,gte=1,lte=65535" json:"port"`
	FontSize    uint16 `gorm:"column:font_size;not null;default:14" form:"font_size" binding:"required,gte=8,lte=48" json:"font_size"`
	Background  string `gorm:"column:background;not null;size:128;default:'#000000'" form:"background" binding:"required,hexcolor" json:"background"`
	Foreground  string `gorm:"column:foreground;not null;size:128;default:'#FFFFFF'" form:"foreground" binding:"required,hexcolor" json:"foreground"`
	CursorColor string `gorm:"column:cursor_color;not null;size:128;default:'#FFFFFF'" form:"cursor_color" binding:"required,hexcolor" json:"cursor_color"`
	FontFamily  string `gorm:"column:font_family;not null;size:128;default:'Courier'" form:"font_family" binding:"min=1,max=128" json:"font_family"`
	CursorStyle string `gorm:"column:cursor_style;not null;size:128;default:'block'" form:"cursor_style" binding:"min=1,max=128" json:"cursor_style"`
	Shell       string `gorm:"column:shell;not null;size:64;default:'sh'" form:"shell" binding:"min=1,max=128" json:"shell"`
	PtyType     string `gorm:"column:pty_type;not null;size:64;default:'xterm-256color'" form:"pty_type" binding:"min=1,max=128" json:"pty_type"`
	InitCmd     string `gorm:"column:init_cmd;type:text" form:"init_cmd" json:"init_cmd"`
	InitBanner  string `gorm:"column:init_banner;type:text" form:"init_banner" json:"init_banner"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c *SshConf) Create(conf *SshConf) error {
	return Db.Create(conf).Error
}

func (c *SshConf) FindByID(id uint, uid uint) (SshConf, error) {
	var conf SshConf
	err := Db.First(&conf, "id = ? AND uid = ?", id, uid).Error
	return conf, err
}

func (c *SshConf) FindAll(offset, limit int, uid uint) ([]SshConf, error) {
	var list []SshConf
	err := Db.Where("uid = ?", uid).Offset(offset).Limit(limit).Order("updated_at desc").Find(&list).Error
	return list, err
}

func (_ *SshConf) UpdateById(id, uid uint, conf *SshConf) error {
	return Db.Model(&conf).Where("id = ? AND uid = ?", id, uid).Select("*").Omit("id", "uid").Updates(conf).Error
}

func (c *SshConf) DeleteByID(id, uid uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND uid = ?", id, uid).Error
}

// BeforeCreate Hook
func (c *SshConf) BeforeCreate(_ *gorm.DB) error {
	_, err := c.sshConfEncrypt()
	return err
}

// BeforeUpdate Hook
func (c *SshConf) BeforeUpdate(_ *gorm.DB) error {
	_, err := c.sshConfEncrypt()
	return err
}

// AfterFind Hook
func (c *SshConf) AfterFind(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.DecryptString(c.Pwd, config.DefaultConfig.AesSecret)
	if err != nil {
		return errors.New("解密密码错误," + err.Error())
	}
	c.CertPwd, err = utils.DecryptString(c.CertPwd, config.DefaultConfig.AesSecret)
	if err != nil {
		return errors.New("解密证书密码错误,," + err.Error())
	}
	c.CertData, err = utils.DecryptString(c.CertData, config.DefaultConfig.AesSecret)
	if err != nil {
		return errors.New("解密密证书数据错误," + err.Error())
	}
	return nil
}

func (c *SshConf) sshConfEncrypt() (*SshConf, error) {
	var err error
	c.Pwd, err = utils.EncryptString(c.Pwd, config.DefaultConfig.AesSecret)
	if err != nil {
		return c, errors.New("加密密码错误," + err.Error())
	}
	c.CertPwd, err = utils.EncryptString(c.CertPwd, config.DefaultConfig.AesSecret)
	if err != nil {
		return c, errors.New("加密证书密码错误," + err.Error())
	}
	c.CertData, err = utils.EncryptString(c.CertData, config.DefaultConfig.AesSecret)
	if err != nil {
		return c, errors.New("加密证书数据错误," + err.Error())
	}
	return c, nil
}
