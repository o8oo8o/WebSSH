package model

import (
	"errors"
	"fmt"
	"gossh/app/config"
	"gossh/app/utils"
	"gossh/gorm"
	"time"
)

type SshdUser struct {
	ID       uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Name     string   `gorm:"column:name;uniqueIndex;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Pwd      string   `gorm:"column:pwd;size:64" form:"pwd" binding:"required,min=1,max=64" json:"pwd"`
	DescInfo string   `gorm:"column:desc_info;size:64" form:"desc_info" binding:"required,min=1,max=64" json:"desc_info"`
	IsEnable string   `gorm:"column:is_enable;not null;size:64;default:'Y'" form:"is_enable" binding:"required,min=1,max=64,oneof=Y N" json:"is_enable"`
	WorkDir  string   `gorm:"column:work_dir;size:4096;default:''" form:"work_dir" binding:"max=4096" json:"work_dir"`
	ExpiryAt DateTime `gorm:"column:expiry_at;not null"  json:"expiry_at"  form:"expiry_at" binding:"required"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c *SshdUser) Create(user *SshdUser) error {
	return Db.Create(user).Error
}

func (c *SshdUser) FindByNameAndPwd(name, pwd string) (SshdUser, error) {
	var user SshdUser
	err := Db.First(&user, "name = ? AND is_enable = ?", name, "Y").Error
	if err != nil {
		return SshdUser{}, err
	}
	if user.Pwd != pwd {
		return SshdUser{}, errors.New("password error")
	}

	if user.ExpiryAt.ToTime().Unix() < time.Now().Unix() {
		return user, errors.New(fmt.Sprintf("sshd user '%s' expired", name))
	}
	return user, err
}

func (c *SshdUser) FindByName(name string) (SshdUser, error) {
	var user SshdUser
	err := Db.Find(&user, "name = ?", name).Error
	return user, err
}

func (c *SshdUser) FindByID(id uint) (SshdUser, error) {
	var user SshdUser
	err := Db.First(&user, "id = ?", id).Error
	return user, err
}

func (c *SshdUser) FindAll(limit, offset int) ([]SshdUser, error) {
	var list []SshdUser
	err := Db.Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

func (c *SshdUser) UpdateById(id uint, user *SshdUser) error {
	return Db.Model(&c).Where("id = ?", id).Select("*").Omit("id").Updates(user).Error
}

func (c *SshdUser) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ?", id).Error
}

// BeforeCreate Hook
func (c *SshdUser) BeforeCreate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.EncryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// BeforeUpdate Hook
func (c *SshdUser) BeforeUpdate(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.EncryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}

// AfterFind Hook
func (c *SshdUser) AfterFind(_ *gorm.DB) error {
	var err error
	c.Pwd, err = utils.DecryptString(c.Pwd, config.DefaultConfig.AesSecret)
	return err
}
