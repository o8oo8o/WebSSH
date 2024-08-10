package model

type PolicyConf struct {
	ID        uint   `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	NetPolicy string `gorm:"column:net_policy;not null;size:64;default:'Y'" form:"net_policy" binding:"required,min=1,max=64,oneof=Y N" json:"net_policy"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c PolicyConf) Create(conf *PolicyConf) error {
	return Db.Create(conf).Error
}

func (c PolicyConf) FindByID(id uint) (PolicyConf, error) {
	var conf PolicyConf
	err := Db.First(&conf, "id = ? ", id).Error
	return conf, err
}

func (c PolicyConf) FindAll(offset, limit int) ([]PolicyConf, error) {
	var list []PolicyConf
	err := Db.Offset(offset).Limit(limit).Order("updated_at desc").Find(&list).Error
	return list, err
}

func (c PolicyConf) UpdateById(id uint, conf *PolicyConf) error {
	return Db.Model(&c).Where("id = ?", id).Updates(conf).Error
}

func (c PolicyConf) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ?", id).Error
}
