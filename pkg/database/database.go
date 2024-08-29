package database

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	logga "github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

type Handler struct {
	DB *gorm.DB
}

func Init(url string) Handler {
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{})
	logga.Warn().Msgf("Connecting to database at %s", url)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}

	err = db.AutoMigrate(
		&models.User{},
		&models.Game{},
		&models.Tag{},
		&models.Like{},
		&models.Comment{},
		&models.Bookmark{},
		&models.Follow{},
		&models.RecentlyPlayed{},
		&models.Genre{},
		&models.UserPreference{},
		&models.GenrePreference{},
		&models.UserGameInteraction{},
		&models.UserSeenGame{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate: %v", err)
	}

	err = AddIndexes(db)
	if err != nil {
		log.Fatalf("Failed to add indexes: %v", err)
	}

	return Handler{DB: db}
}

func AddIndexes(db *gorm.DB) error {
	//if err := db.Exec("CREATE INDEX idx_user_seen_games_user_id_game_id_seen_at ON user_seen_games(user_id, game_id, seen_at)").Error; err != nil {
	//	return err
	//}
	//
	//if err := db.Exec("CREATE INDEX idx_games_genre_id_play_count ON games(genre_id, play_count)").Error; err != nil {
	//	return err
	//}
	//
	//if err := db.Exec("CREATE INDEX idx_games_play_time ON games(play_time)").Error; err != nil {
	//	return err
	//}
	//
	//if err := db.Exec("CREATE INDEX idx_user_game_interactions_game_id ON user_game_interactions(game_id)").Error; err != nil {
	//	return err
	//}

	return nil
}
