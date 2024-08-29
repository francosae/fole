package models

import "time"

type Comment struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Content   string
	CreatedAt time.Time `gorm:"default:current_timestamp"`
	UserID    string
	User      User `gorm:"foreignKey:UserID"`
	GameID    string
	Game      Game      `gorm:"foreignKey:GameID"`
	ParentID  *string   `gorm:"type:uuid;null"`
	Parent    *Comment  `gorm:"foreignKey:ParentID"`
	Replies   []Comment `gorm:"foreignKey:ParentID"`
	IsDeleted bool      `gorm:"default:false"` // soft delete baby

}
