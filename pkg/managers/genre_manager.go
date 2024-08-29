package managers

import (
	"fmt"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"github.com/rs/zerolog/log"
)

type GenreManager struct {
	db database.Handler
}

func NewGenreManager(db database.Handler) *GenreManager {
	return &GenreManager{db: db}
}

func (gm *GenreManager) EnsureGenres() error {
	genres := []string{
		"Action", "Adventure", "Arcade", "Puzzle", "Strategy",
		"RPG", "Simulation", "Sports", "Racing", "Shooter",
	}

	for _, genreName := range genres {
		var genre models.Genre
		result := gm.db.DB.Where(models.Genre{Name: genreName}).FirstOrCreate(&genre)
		if result.Error != nil {
			return fmt.Errorf("failed to ensure genre %s: %v", genreName, result.Error)
		}
		if result.RowsAffected > 0 {
			log.Printf("Created new genre: %s", genreName)
		}
	}

	return nil
}

func (gm *GenreManager) ListGenres() ([]models.Genre, error) {
	var genres []models.Genre
	result := gm.db.DB.Find(&genres)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list genres: %v", result.Error)
	}
	return genres, nil
}

func (gm *GenreManager) AddGenre(name string) error {
	genre := models.Genre{Name: name}
	result := gm.db.DB.Create(&genre)
	if result.Error != nil {
		return fmt.Errorf("failed to add genre %s: %v", name, result.Error)
	}
	log.Printf("Added new genre: %s", name)
	return nil
}

func (gm *GenreManager) UpdateGameGenre(gameID string, genreName string) error {
	var genre models.Genre
	if err := gm.db.DB.Where("name = ?", genreName).First(&genre).Error; err != nil {
		return fmt.Errorf("genre %s not found: %v", genreName, err)
	}

	result := gm.db.DB.Model(&models.Game{}).Where("id = ?", gameID).Update("genre_id", genre.ID)
	if result.Error != nil {
		return fmt.Errorf("failed to update game genre: %v", result.Error)
	}
	log.Printf("Updated game %s with genre %s", gameID, genreName)
	return nil
}
