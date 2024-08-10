package model

type CmdNote struct {
	ID      uint   `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Uid     uint   `gorm:"column:uid;not null;default:0" form:"uid" json:"uid"`
	CmdName string `gorm:"column:cmd_name;type:text" form:"cmd_name" binding:"required" json:"cmd_name"`
	CmdData string `gorm:"column:cmd_data;type:text" form:"cmd_data" binding:"required" json:"cmd_data"`

	CreatedAt DateTime `gorm:"created_at" json:"-"`
	UpdatedAt DateTime `gorm:"updated_at" json:"-"`
}

func (c CmdNote) Create(cmd *CmdNote) error {
	return Db.Create(cmd).Error
}

func (c CmdNote) FindByName(name string) (CmdNote, error) {
	var cmd CmdNote
	err := Db.First(&cmd, "name = ?", name).Error
	return cmd, err
}

func (c CmdNote) FindByID(id uint, uid uint) (CmdNote, error) {
	var cmd CmdNote
	err := Db.First(&cmd, "id = ? AND uid = ?", id, uid).Error
	return cmd, err
}

func (c CmdNote) FindAll(offset, limit int, uid uint) ([]CmdNote, error) {
	var list []CmdNote
	err := Db.Where("uid = ?", uid).Offset(offset).Limit(limit).Order("updated_at desc").Find(&list).Error
	return list, err
}

func (c CmdNote) UpdateById(id, uid uint, cmd *CmdNote) error {
	return Db.Model(&c).Where("id = ? AND uid = ?", id, uid).Updates(cmd).Error
}

func (c CmdNote) DeleteByID(id, uid uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND uid = ?", id, uid).Error
}
