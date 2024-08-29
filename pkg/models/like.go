package models

import "time"

type Like struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"default:current_timestamp"`
	UserID    string
	User      User `gorm:"foreignKey:UserID"`
	GameID    string
	Game      Game `gorm:"foreignKey:GameID"`
}

func (Like) TableName() string {
	return "likes"
}
