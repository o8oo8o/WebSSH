package model

import "time"

type NetFilter struct {
	ID        uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Name      string   `gorm:"column:name;not null" form:"name" json:"name" binding:"required"`
	Cidr      string   `gorm:"column:cidr;not null" form:"cidr" json:"cidr" binding:"required,cidr"`
	NetPolicy string   `gorm:"column:net_policy;not null;size:64;default:'Y'" form:"net_policy" binding:"required,min=1,max=64,oneof=Y N" json:"net_policy"`
	PolicyNo  uint     `gorm:"column:policy_no;not null;" form:"policy_no" json:"policy_no" binding:"required,gte=1,lte=65535"`
	ExpiryAt  DateTime `gorm:"column:expiry_at;not null"  json:"expiry_at"  form:"expiry_at" binding:"required"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c NetFilter) Create(filter *NetFilter) error {
	return Db.Create(filter).Error
}

func (c NetFilter) FindByName(name string) (NetFilter, error) {
	var filter NetFilter
	err := Db.First(&filter, "name = ?", name).Error
	return filter, err
}

func (c NetFilter) FindByID(id uint) (NetFilter, error) {
	var filter NetFilter
	err := Db.First(&filter, "id = ? ", id).Error
	return filter, err
}

func (c NetFilter) FindAll(offset, limit int) ([]NetFilter, error) {
	var list []NetFilter
	err := Db.Offset(offset).Limit(limit).Order("policy_no asc, expiry_at, updated_at desc").Find(&list).Error
	return list, err
}

func (c NetFilter) FindAllPolicy(policy string) ([]NetFilter, error) {
	var list []NetFilter
	err := Db.Where("net_policy = ? AND expiry_at > ?", policy, time.Now()).Order("policy_no asc, expiry_at, updated_at desc").Find(&list).Error
	return list, err
}

func (c NetFilter) UpdateById(id uint, filter *NetFilter) error {
	return Db.Model(&c).Where("id = ?", id).Updates(filter).Error
}

func (c NetFilter) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ?", id).Error
}
