package models

import (
	"gorm.io/gorm"
	"time"
)

type Game struct {
	ID                string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Title             string
	Description       string
	PlayCount         int       `gorm:"default:0"`
	PlayTime          time.Time `gorm:"default:0"`
	LikeCount         int       `gorm:"default:0"`
	CommentCount      int       `gorm:"default:0"`
	BookmarkCount     int       `gorm:"default:0"`
	IsFeatured        bool      `gorm:"default:false"`
	GenreID           string    `gorm:"type:uuid"`
	Genre             Genre     `gorm:"foreignKey:GenreID"`
	ButtonMapping     bool      `gorm:"default:false"`
	EmbedLink         string
	GameType          string
	ThumbnailFileName string
	IsLandscape       bool
	IsClaimed         bool `gorm:"default:false"`
	CreatorID         *string
	Creator           *User `gorm:"foreignKey:CreatorID"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
	IsDeleted         bool                  `gorm:"default:false"`
	SeenByUsers       []UserSeenGame        `gorm:"foreignKey:GameID"`
	Interactions      []UserGameInteraction `gorm:"foreignKey:GameID"`
	Tags              []Tag                 `gorm:"many2many:game_tags;"`
}

type Tag struct {
	gorm.Model
	Name  string `gorm:"uniqueIndex"`
	Games []Game `gorm:"many2many:game_tags;"`
}

type UserSeenGame struct {
	UserID    string    `gorm:"primaryKey;type:varchar(255)"`
	GameID    string    `gorm:"primaryKey;type:uuid"`
	SeenAt    time.Time `gorm:"autoCreateTime"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type Genre struct {
	ID   string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name string `gorm:"uniqueIndex"`
}

type UserGameInteraction struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          string `gorm:"type:varchar(255)"`
	GameID          string `gorm:"type:uuid"`
	PlayCount       int    `gorm:"default:0"`
	PlayTime        int    `gorm:"default:0"`
	LikeCount       int    `gorm:"default:0"`
	CommentCount    int    `gorm:"default:0"`
	BookmarkCount   int    `gorm:"default:0"`
	LastInteraction time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	User User `gorm:"foreignKey:UserID"`
	Game Game `gorm:"foreignKey:GameID"`
}

type UserPreference struct {
	ID          uint              `gorm:"primaryKey"`
	UserID      string            `gorm:"type:varchar(255);uniqueIndex"`
	Preferences []GenrePreference `gorm:"foreignKey:UserPreferenceID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	User User `gorm:"foreignKey:UserID"`
}

type GenrePreference struct {
	ID               uint `gorm:"primaryKey"`
	UserPreferenceID uint
	GenreID          string  `gorm:"type:uuid"`
	Preference       float64 `gorm:"type:decimal(3,2)"`
}

func (game *Game) BeforeCreate(tx *gorm.DB) error {
	game.CreatedAt = time.Now()
	game.UpdatedAt = time.Now()
	return nil
}

func (game *Game) BeforeUpdate(tx *gorm.DB) error {
	game.UpdatedAt = time.Now()
	return nil
}
