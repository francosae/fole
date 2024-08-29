package models

import "time"

type Follow struct {
	ID          string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	CreatedAt   time.Time `gorm:"default:current_timestamp"`
	FollowerID  string
	Follower    User `gorm:"foreignKey:FollowerID"`
	FollowingID string
	Following   User `gorm:"foreignKey:FollowingID"`
}

func (Follow) TableName() string {
	return "follows"
}
