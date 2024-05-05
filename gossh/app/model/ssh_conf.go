package model

type SshConf struct {
	ID          uint     `gorm:"primaryKey,autoIncrement" form:"id" json:"id"`
	Uid         uint     `gorm:"not null;default:0" form:"uid" json:"uid"`
	Name        string   `gorm:"not null;size:64" form:"name" binding:"required,min=1,max=63" json:"name"`
	Address     string   `gorm:"size:128" form:"address" binding:"required,min=1,max=128" json:"address"`
	User        string   `gorm:"size:128" form:"user" binding:"required,min=1,max=128" json:"user"`
	Pwd         string   `gorm:"not null;size:128;default:''" form:"pwd" binding:"max=128" json:"pwd"`
	AuthType    string   `gorm:"not null;size:32;default:'pwd'" form:"auth_type" binding:"required,min=1,max=32,oneof=pwd cert" json:"auth_type"`
	NetType     string   `gorm:"not null;size:32;default:'tcp4'" form:"net_type" binding:"required,min=1,max=32,oneof=tcp4 tcp6" json:"net_type"`
	CertData    string   `gorm:"type:text" form:"cert_data" json:"cert_data"`
	CertPwd     string   `gorm:"not null;size:128;default:''" form:"cert_pwd" binding:"max=128" json:"cert_pwd"`
	Port        uint16   `gorm:"not null;default:22" form:"port" binding:"required,gte=1,lte=65535" json:"port"`
	FontSize    uint16   `gorm:"not null;default:14" form:"font_size" binding:"required,gte=8,lte=48" json:"font_size"`
	Background  string   `gorm:"not null;size:128;default:'#000000'" form:"background" binding:"required,hexcolor" json:"background"`
	Foreground  string   `gorm:"not null;size:128;default:'#FFFFFF'" form:"foreground" binding:"required,hexcolor" json:"foreground"`
	CursorColor string   `gorm:"not null;size:128;default:'#FFFFFF'" form:"cursor_color" binding:"required,hexcolor" json:"cursor_color"`
	FontFamily  string   `gorm:"not null;size:128;default:'Courier'" form:"font_family" binding:"min=1,max=128" json:"font_family"`
	CursorStyle string   `gorm:"not null;size:128;default:'block'" form:"cursor_style" binding:"min=1,max=128" json:"cursor_style"`
	Shell       string   `gorm:"not null;size:64;default:'bash'" form:"shell" binding:"min=1,max=128" json:"shell"`
	PtyType     string   `gorm:"not null;size:64;default:'xterm-256color'" form:"pty_type" binding:"min=1,max=128" json:"pty_type"`
	InitCmd     string   `gorm:"type:text" form:"init_cmd" json:"init_cmd"`
	InitBanner  string   `gorm:"type:text" form:"init_banner" json:"init_banner"`
	CreatedAt   DateTime `gorm:"created_at" json:"-"`
	UpdatedAt   DateTime `gorm:"updated_at" json:"-"`
}

func (c SshConf) Create(conf *SshConf) error {
	return Db.Create(conf).Error
}

func (c SshConf) FindByID(id uint, uid uint) (SshConf, error) {
	var conf SshConf
	err := Db.First(&conf, "id = ? AND uid = ?", id, uid).Error
	return conf, err
}

func (c SshConf) FindAll(offset, limit int, uid uint) ([]SshConf, error) {
	var list []SshConf
	err := Db.Where("uid = ?", uid).Offset(offset).Limit(limit).Order("updated_at desc").Find(&list).Error
	return list, err
}

func (c SshConf) UpdateById(id, uid uint, conf *SshConf) error {
	return Db.Model(&c).Where("id = ? AND uid = ?", id, uid).Updates(conf).Error
}

func (c SshConf) DeleteByID(id, uid uint) error {
	return Db.Unscoped().Delete(&c, "id = ? AND uid = ?", id, uid).Error
}
