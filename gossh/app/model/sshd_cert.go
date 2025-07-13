package model

import "time"

type SshdCert struct {
	ID       uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	DescInfo string   `gorm:"column:desc_info;size:64" form:"desc_info" binding:"required,min=1,max=64" json:"desc_info"`
	Name     string   `gorm:"column:name;uniqueIndex;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	PubKey   string   `gorm:"column:pub_key;type:text;not null;" form:"pub_key" binding:"required,min=16" json:"pub_key"`
	IsEnable string   `gorm:"column:is_enable;not null;size:64;default:'Y'" form:"is_enable" binding:"required,min=1,max=64,oneof=Y N" json:"is_enable"`
	ExpiryAt DateTime `gorm:"column:expiry_at;not null"  json:"expiry_at"  form:"expiry_at" binding:"required"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c *SshdCert) Create(user *SshdCert) error {
	return Db.Create(user).Error
}

func (c *SshdCert) FindByName(name string) (SshdCert, error) {
	var user SshdCert
	err := Db.Find(&user, "name = ?", name).Error
	return user, err
}

func (c *SshdCert) FindByID(id uint) (SshdCert, error) {
	var user SshdCert
	err := Db.First(&user, "id = ?", id).Error
	return user, err
}

func (c *SshdCert) FindAll(limit, offset int) ([]SshdCert, error) {
	var list []SshdCert
	err := Db.Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

func (c *SshdCert) GetAuthorizedKeys() (string, error) {
	var authorizedKeys = ""
	var list []string
	err := Db.Model(c).Select("pub_key").Where("is_enable = ? AND expiry_at > ?", "Y", time.Now()).Find(&list).Error
	if err != nil {
		return "", err
	}

	for _, v := range list {
		authorizedKeys += v + "\n"
	}
	return authorizedKeys, err
}

func (c *SshdCert) UpdateById(id uint, user *SshdCert) error {
	return Db.Model(&c).Where("id = ?", id).Select("*").Omit("id").Updates(user).Error
}

func (c *SshdCert) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ?", id).Error
}
