package model

type SshUser struct {
	ID       uint     `gorm:"primaryKey,autoIncrement" form:"id" json:"id"`
	Name     string   `gorm:"uniqueIndex;not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Pwd      string   `gorm:"size:64" form:"pwd" binding:"required,min=1,max=64" json:"pwd"`
	DescInfo string   `gorm:"size:64" form:"desc_info" binding:"required,min=1,max=64" json:"desc_info"`
	IsAdmin  string   `gorm:"not null;size:64;default:'N'" form:"is_admin" binding:"required,min=1,max=64,oneof=Y N" json:"is_admin"`
	IsEnable string   `gorm:"not null;size:64;default:'Y'" form:"is_enable" binding:"required,min=1,max=64,oneof=Y N" json:"is_enable"`
	IsRoot   string   `gorm:"not null;size:64;default:'N'" form:"is_root"  json:"is_root"`
	ExpiryAt DateTime `gorm:"expiry_at;not null"  json:"expiry_at"  form:"expiry_at" binding:"required"`

	CreatedAt DateTime `gorm:"created_at" json:"-"`
	UpdatedAt DateTime `gorm:"updated_at" json:"-"`
}

func (c SshUser) Create(user *SshUser) error {
	return Db.Create(user).Error
}

func (c SshUser) FindByNameAndPwd(name, pwd string) (SshUser, error) {
	var user SshUser
	err := Db.First(&user, "name = ? AND pwd = ?", name, pwd).Error
	return user, err
}

func (c SshUser) FindByName(name string) (SshUser, error) {
	var user SshUser
	err := Db.Find(&user, "name = ?", name).Error
	return user, err
}

func (c SshUser) FindByID(id uint) (SshUser, error) {
	var user SshUser
	err := Db.First(&user, "id = ?", id).Error
	return user, err
}

func (c SshUser) FindAll(limit, offset int) ([]SshUser, error) {
	var list []SshUser
	err := Db.Where("is_root = ?", "N").Limit(limit).Offset(offset).Find(&list).Error
	return list, err
}

func (c SshUser) UpdateByName(name string, user *SshUser) error {
	return Db.Model(&c).Where("name = ?", name).Updates(user).Error
}

func (c SshUser) UpdateById(id uint, user *SshUser) error {
	return Db.Model(&c).Where("id = ? AND is_root = ?", id, "N").Updates(user).Error
}

func (c SshUser) UpdatePassword(id uint, user *SshUser) error {
	return Db.Model(&c).Where("id = ?", id).Updates(user).Error
}

func (c SshUser) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND is_root = ?", id, "N").Error
}
