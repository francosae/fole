package models

import "time"

type RecentlyPlayed struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	LastPlayedAt time.Time `gorm:"default:current_timestamp"`
	PlayCount    int       `gorm:"default:1"`
	UserID       string
	User         User `gorm:"foreignKey:UserID"`
	GameID       string
	Game         Game `gorm:"foreignKey:GameID"`
}

func (RecentlyPlayed) TableName() string {
	return "recently_played"
}
