package model

import (
	"gossh/app/config"
	"gossh/app/utils"
	"gossh/gorm"
)

type SshUser struct {
	ID       uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Name     string   `gorm:"column:name;uniqueIndex;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Pwd      string   `gorm:"column:pwd;size:64" form:"pwd" binding:"required,min=1,max=64" json:"pwd"`
	DescInfo string   `gorm:"column:desc_info;size:64" form:"desc_info" binding:"required,min=1,max=64" json:"desc_info"`
	IsAdmin  string   `gorm:"column:is_admin;not null;size:64;default:'N'" form:"is_admin" binding:"required,min=1,max=64,oneof=Y N" json:"is_admin"`
	IsEnable string   `gorm:"column:is_enable;not null;size:64;default:'Y'" form:"is_enable" binding:"required,min=1,max=64,oneof=Y N" json:"is_enable"`
	IsRoot   string   `gorm:"column:is_root;not null;size:64;default:'N'" form:"is_root"  json:"is_root"`
	ExpiryAt DateTime `gorm:"column:expiry_at;not null"  json:"expiry_at"  form:"expiry_at" binding:"required"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c *SshUser) Create(user *SshUser) error {
	return Db.Create(user).Error
}

func (c *SshUser) FindByNameAndPwd(name, pwd string) (SshUser, error) {
	var user SshUser
	encryptPwd, err := utils.AesEncrypt(pwd, config.DefaultConfig.AesSecret)
	if err != nil {
		return SshUser{}, err
	}
	err = Db.First(&user, "name = ? AND pwd = ?", name, encryptPwd).Error
	return user, err
}

func (c *SshUser) FindByName(name string) (SshUser, error) {
	var user SshUser
	err := Db.Find(&user, "name = ?", name).Error
	return user, err
}

func (c *SshUser) FindByID(id uint) (SshUser, error) {
	var user SshUser
	err := Db.First(&user, "id = ?", id).Error
	return user, err
}

func (c *SshUser) FindAll(limit, offset int) ([]SshUser, error) {
	var list []SshUser
	err := Db.Where("is_root = ?", "N").Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

func (c *SshUser) UpdateById(id uint, user *SshUser) error {
	return Db.Model(&c).Where("id = ? AND is_root = ?", id, "N").Select("*").Omit("id", "is_root").Updates(user).Error
}

func (c *SshUser) UpdatePassword(id uint, user *SshUser) error {
	return Db.Model(&c).Where("id = ?", id).Select("*").Omit("id").Updates(user).Error
}

func (c *SshUser) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND is_root = ?", id, "N").Error
}

// BeforeCreate Hook
func (c *SshUser) BeforeCreate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.AesEncrypt(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// BeforeUpdate Hook
func (c *SshUser) BeforeUpdate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.AesEncrypt(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// AfterFind Hook
func (c *SshUser) AfterFind(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.AesDecrypt(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}
