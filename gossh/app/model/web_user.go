package model

import (
	"errors"
	"gossh/app/config"
	"gossh/app/utils"
	"gossh/gorm"
)

type WebUser struct {
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

func (c *WebUser) Create(user *WebUser) error {
	return Db.Create(user).Error
}

func (c *WebUser) FindByNameAndPwd(name, pwd string) (WebUser, error) {
	var user WebUser
	err := Db.First(&user, "name = ?", name).Error
	if err != nil {
		return WebUser{}, err
	}
	if user.Pwd != pwd {
		return WebUser{}, errors.New("password error")
	}
	return user, err
}

func (c *WebUser) FindByName(name string) (WebUser, error) {
	var user WebUser
	err := Db.Find(&user, "name = ?", name).Error
	return user, err
}

func (c *WebUser) FindByID(id uint) (WebUser, error) {
	var user WebUser
	err := Db.First(&user, "id = ?", id).Error
	return user, err
}

func (c *WebUser) FindAll(limit, offset int) ([]WebUser, error) {
	var list []WebUser
	err := Db.Where("is_root = ?", "N").Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

func (c *WebUser) UpdateById(id uint, user *WebUser) error {
	return Db.Model(&c).Where("id = ? AND is_root = ?", id, "N").Select("*").Omit("id", "is_root").Updates(user).Error
}

func (c *WebUser) UpdatePassword(id uint, user *WebUser) error {
	return Db.Model(&c).Where("id = ?", id).Select("*").Omit("id").Updates(user).Error
}

func (c *WebUser) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND is_root = ?", id, "N").Error
}

// BeforeCreate Hook
func (c *WebUser) BeforeCreate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.EncryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// BeforeUpdate Hook
func (c *WebUser) BeforeUpdate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.EncryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// AfterFind Hook
func (c *WebUser) AfterFind(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.DecryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}
