package models

import "time"

type Bookmark struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"default:current_timestamp"`
	UserID    string
	User      User `gorm:"foreignKey:UserID"`
	GameID    string
	Game      Game `gorm:"foreignKey:GameID"`
}

func (Bookmark) TableName() string {
	return "bookmarks"
}
