package model

type LoginAudit struct {
	ID        uint     `gorm:"column:id;primaryKey,autoIncrement" form:"id" json:"id"`
	Name      string   `gorm:"column:name;not null;size:128" form:"name" binding:"required,min=1,max=128" json:"name"`
	Pwd       string   `gorm:"column:pwd;size:128" form:"pwd" binding:"required,min=1,max=128" json:"pwd"`
	ClientIp  string   `gorm:"column:client_ip;size:128" form:"client_ip" binding:"required,min=1,max=128" json:"client_ip"`
	UserAgent string   `gorm:"column:user_agent;size:512" form:"user_agent" binding:"required,min=0,max=512" json:"user_agent"`
	ErrMsg    string   `gorm:"column:err_msg;size:64" form:"err_msg" binding:"required,min=1,max=64" json:"err_msg"`
	IsSuccess string   `gorm:"column:is_success;not null;size:64;default:'N'" form:"is_success" binding:"required,min=1,max=64,oneof=Y N" json:"is_success"`
	OccurAt   DateTime `gorm:"column:occur_at;not null"  json:"occur_at"  form:"occur_at" binding:"required"`

	CreatedAt DateTime `gorm:"column:created_at" json:"-"`
	UpdatedAt DateTime `gorm:"column:updated_at" json:"-"`
}

func (c LoginAudit) Create(audit *LoginAudit) error {
	return Db.Create(audit).Error
}

func (c LoginAudit) Search(
	isSuccess, name, clientIp string,
	occurBegin, occurEnd DateTime, offset, limit int,
) ([]LoginAudit, int64, error) {
	var list []LoginAudit
	var db = Db
	if isSuccess != "" {
		db = db.Where("is_success = ?", isSuccess)
	}
	if name != "" {
		db = db.Where("name like ?", "%"+name+"%")
	}
	if clientIp != "" {
		db = db.Where("client_ip = ?", clientIp)
	}

	if occurBegin.String() != "0001-01-01 00:00:00" && occurEnd.String() != "0001-01-01 00:00:00" {
		db = db.Where("occur_at between  ? AND ?", occurBegin, occurEnd)
	}
	var count int64
	err := db.Model(&LoginAudit{}).Count(&count).Error
	if err != nil {
		return list, count, err
	}
	return list, count, db.Debug().Order("occur_at desc").Offset(offset).Limit(limit).Find(&list).Error
}

func (c LoginAudit) FindByID(id uint) (LoginAudit, error) {
	var audit LoginAudit
	err := Db.First(&audit, "id = ?", id).Error
	return audit, err
}

func (c LoginAudit) DeleteByID(id uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND is_root = ?", id, "N").Error
}
