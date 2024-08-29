package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	UID             string `gorm:"primaryKey"`
	Email           string `gorm:"unique"`
	Username        string `gorm:"unique"`
	DisplayName     *string
	ProfileImageURL *string
	Bio             *string
	Gender          *string
	Birthday        *time.Time
	CreatedAt       time.Time `gorm:"default:current_timestamp"`
	UpdatedAt       time.Time
	FollowersCount  int              `gorm:"default:0"`
	FollowingCount  int              `gorm:"default:0"`
	Tags            []Tag            `gorm:"many2many:game_tags;"`
	Games           []Game           `gorm:"foreignKey:CreatorID"`
	Likes           []Like           `gorm:"foreignKey:UserID"`
	Comments        []Comment        `gorm:"foreignKey:UserID"`
	Bookmarks       []Bookmark       `gorm:"foreignKey:UserID"`
	FollowedBy      []Follow         `gorm:"foreignKey:FollowingID"`
	Following       []Follow         `gorm:"foreignKey:FollowerID"`
	RecentlyPlayed  []RecentlyPlayed `gorm:"foreignKey:UserID"`
}

func (user *User) BeforeCreate(tx *gorm.DB) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return nil
}

func (user *User) BeforeUpdate(tx *gorm.DB) error {
	user.UpdatedAt = time.Now()
	return nil
}
