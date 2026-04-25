package models

type AllowlistEntry struct {
	ID       int    `gorm:"primaryKey,autoIncrement,column:id"`
	Platform string `gorm:"index:idx_platform_user,unique,column:platform"`
	UserID   string `gorm:"index:idx_platform_user,unique,column:user_id"`
}

func (AllowlistEntry) TableName() string {
	return "allowlist"
}
