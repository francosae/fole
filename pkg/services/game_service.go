package services

import (
	"errors"
	"fmt"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/supabase"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"gorm.io/gorm"
	"time"
)

type GameService struct {
	databaseHandler database.Handler
	supabaseAuth    *supabase.SupabaseAuth
	algoliaClient   *search.Client
}

func NewGameService(databaseHandler database.Handler, supabaseAuth *supabase.SupabaseAuth, algoliaClient *search.Client) *GameService {
	return &GameService{
		databaseHandler: databaseHandler,
		supabaseAuth:    supabaseAuth,
		algoliaClient:   algoliaClient,
	}
}

func (gs *GameService) GameDetailsByGameId(gameId string) (game models.Game, err error) {
	err = gs.databaseHandler.DB.Where("id = ?", gameId).First(&game).Error
	return game, err
}

func (gs *GameService) CreateInteractionByGameId(gameId string, userId string, interactionType string) (err error) {
	tx := gs.databaseHandler.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var game models.Game
	if err := tx.First(&game, "id = ?", gameId).Error; err != nil {
		tx.Rollback()
		return err
	}

	switch interactionType {
	case "like":
		like := models.Like{
			GameID: gameId,
			UserID: userId,
		}
		if err := tx.Create(&like).Error; err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Model(&game).Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return err
		}

	case "bookmark":
		bookmark := models.Bookmark{
			GameID: gameId,
			UserID: userId,
		}
		if err := tx.Create(&bookmark).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Model(&game).Update("bookmark_count", gorm.Expr("bookmark_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return err
		}

	case "play":
		var recentlyPlayed models.RecentlyPlayed
		result := tx.Where("game_id = ? AND user_id = ?", gameId, userId).First(&recentlyPlayed)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				recentlyPlayed = models.RecentlyPlayed{
					GameID:    gameId,
					UserID:    userId,
					PlayCount: 1,
				}
				if err := tx.Create(&recentlyPlayed).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				tx.Rollback()
				return result.Error
			}
		} else {
			if err := tx.Model(&recentlyPlayed).Updates(map[string]interface{}{
				"last_played_at": time.Now(),
				"play_count":     gorm.Expr("play_count + ?", 1),
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}

		if err := tx.Model(&game).Update("play_count", gorm.Expr("play_count + ?", 1)).Error; err != nil {
			tx.Rollback()
			return err
		}

	default:
		tx.Rollback()
		return fmt.Errorf("invalid interaction type: %s", interactionType)
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}
