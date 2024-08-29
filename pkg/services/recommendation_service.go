package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"sort"
	"time"
)

const (
	seenGameThreshold               = 3 * 24 * time.Hour
	cacheExpirationTime             = 1 * time.Hour
	recommendationCacheKey          = "user:%s:recommendations"
	fallbackRecommendationsCacheKey = "fallback:recommendations"
	maxRecommendations              = 25
)

type RecommendationService struct {
	db          *gorm.DB
	redisClient *redis.Client
}

func NewRecommendationService(databaseHandler database.Handler, redisClient *redis.Client) *RecommendationService {
	return &RecommendationService{
		db:          databaseHandler.DB,
		redisClient: redisClient,
	}
}

func (rs *RecommendationService) GetRecommendations(userId string, page, limit int) ([]models.Game, int64, error) {
	cacheKey := fmt.Sprintf(recommendationCacheKey, userId)
	return rs.getRecommendationsFromCacheOrGenerate(cacheKey, func() ([]models.Game, error) {
		return rs.generatePersonalizedRecommendations(userId)
	}, page, limit)
}

func (rs *RecommendationService) GetFallbackRecommendations(page, limit int) ([]models.Game, int64, error) {
	cacheKey := fmt.Sprintf("%s:%d:%d", fallbackRecommendationsCacheKey, page, limit)
	return rs.getRecommendationsFromCacheOrGenerate(cacheKey, func() ([]models.Game, error) {
		return rs.generateFallbackRecommendations(page, limit)
	}, page, limit)
}

func (rs *RecommendationService) getRecommendationsFromCacheOrGenerate(cacheKey string, generateFunc func() ([]models.Game, error), page, limit int) ([]models.Game, int64, error) {
	cachedRecommendations, err := rs.redisClient.LRange(context.Background(), cacheKey, 0, -1).Result()
	if err == nil && len(cachedRecommendations) > 0 {
		var games []models.Game
		for _, gameJSON := range cachedRecommendations {
			var game models.Game
			if err := json.Unmarshal([]byte(gameJSON), &game); err == nil {
				games = append(games, game)
			}
		}
		return games, int64(len(games)), nil
	}

	recommendations, err := generateFunc()
	if err != nil {
		return nil, 0, err
	}

	rs.cacheRecommendations(cacheKey, recommendations)
	return recommendations, int64(len(recommendations)), nil
}

func (rs *RecommendationService) generatePersonalizedRecommendations(userId string) ([]models.Game, error) {
	var userInteractions []models.UserGameInteraction
	if err := rs.db.Where("user_id = ?", userId).Find(&userInteractions).Error; err != nil {
		return nil, err
	}

	genreScores := make(map[string]float64)
	for _, interaction := range userInteractions {
		var game models.Game
		if err := rs.db.Preload("Genre").Where("id = ?", interaction.GameID).First(&game).Error; err != nil {
			continue
		}
		score := float64(interaction.PlayCount*3 + interaction.PlayTime/60 + interaction.LikeCount*2 + interaction.BookmarkCount*1)
		genreScores[game.GenreID] += score
	}

	var sortedGenres []struct {
		genreID string
		score   float64
	}
	for genreID, score := range genreScores {
		sortedGenres = append(sortedGenres, struct {
			genreID string
			score   float64
		}{genreID, score})
	}
	sort.Slice(sortedGenres, func(i, j int) bool {
		return sortedGenres[i].score > sortedGenres[j].score
	})

	var recommendations []models.Game
	seenThreshold := time.Now().Add(-seenGameThreshold)

	for _, gs := range sortedGenres {
		var genreGames []models.Game
		if err := rs.db.Where("genre_id = ?", gs.genreID).
			Where("id NOT IN (SELECT game_id FROM user_seen_games WHERE user_id = ? AND seen_at > ?)", userId, seenThreshold).
			Order("play_count DESC, like_count DESC").
			Limit(10).
			Find(&genreGames).Error; err != nil {
			continue
		}
		recommendations = append(recommendations, genreGames...)
		if len(recommendations) >= maxRecommendations/2 {
			break
		}
	}

	if len(recommendations) < maxRecommendations {
		mixedPopularGames, err := rs.getMixedPopularGames(userId, maxRecommendations-len(recommendations))
		if err != nil {
			return nil, err
		}
		recommendations = append(recommendations, mixedPopularGames...)
	}

	return recommendations, nil
}

func (rs *RecommendationService) getMixedPopularGames(userId string, numRecommendations int) ([]models.Game, error) {
	var games []models.Game
	seenThreshold := time.Now().Add(-seenGameThreshold)

	err := rs.db.Raw(`
		SELECT g.* 
		FROM games g
		LEFT JOIN (
			SELECT game_id, SUM(play_time) as total_play_time, SUM(play_count) as total_play_count, SUM(like_count) as total_like_count
			FROM user_game_interactions
			GROUP BY game_id
		) ugi ON g.id = ugi.game_id
		WHERE g.id NOT IN (
			SELECT game_id 
			FROM user_seen_games 
			WHERE user_id = ? AND seen_at > ?
		)
		ORDER BY 
			(g.play_count + COALESCE(ugi.total_play_count, 0)) * 0.4 + 
			(g.like_count + COALESCE(ugi.total_like_count, 0)) * 0.3 + 
			COALESCE(ugi.total_play_time, 0) * 0.2 - 
			EXTRACT(EPOCH FROM (NOW() - g.created_at)) / 86400 * 0.1 DESC
		LIMIT ?
	`, userId, seenThreshold, numRecommendations).Scan(&games).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get mixed popular games: %v", err)
	}

	return games, nil
}

func (rs *RecommendationService) generateFallbackRecommendations(page, limit int) ([]models.Game, error) {
	var games []models.Game
	offset := (page - 1) * limit

	err := rs.db.Raw(`
		SELECT g.* 
		FROM games g
		LEFT JOIN (
			SELECT game_id, SUM(play_time) as total_play_time, SUM(play_count) as total_play_count, SUM(like_count) as total_like_count
			FROM user_game_interactions
			GROUP BY game_id
		) ugi ON g.id = ugi.game_id
		ORDER BY 
			(g.play_count + COALESCE(ugi.total_play_count, 0)) * 0.4 + 
			(g.like_count + COALESCE(ugi.total_like_count, 0)) * 0.3 + 
			COALESCE(ugi.total_play_time, 0) * 0.2 - 
			EXTRACT(EPOCH FROM (NOW() - g.created_at)) / 86400 * 0.1 DESC
		OFFSET ?
		LIMIT ?
	`, offset, limit).Scan(&games).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get fallback recommendations: %v", err)
	}

	return games, nil
}

func (rs *RecommendationService) cacheRecommendations(cacheKey string, recommendations []models.Game) {
	var gameJSONs []interface{}
	for _, game := range recommendations {
		gameJSON, _ := json.Marshal(game)
		gameJSONs = append(gameJSONs, gameJSON)
	}
	rs.redisClient.RPush(context.Background(), cacheKey, gameJSONs...)
	rs.redisClient.Expire(context.Background(), cacheKey, cacheExpirationTime)
}

func (rs *RecommendationService) paginateAndReturnGames(games interface{}, page, limit int) ([]models.Game, int64, error) {
	var allGames []models.Game
	switch v := games.(type) {
	case []models.Game:
		allGames = v
	case []string:
		for _, gameJSON := range v {
			var game models.Game
			if err := json.Unmarshal([]byte(gameJSON), &game); err == nil {
				allGames = append(allGames, game)
			}
		}
	}

	totalGames := int64(len(allGames))
	start := (page - 1) * limit
	end := page * limit
	if start >= len(allGames) {
		return []models.Game{}, totalGames, nil
	}
	if end > len(allGames) {
		end = len(allGames)
	}
	return allGames[start:end], totalGames, nil
}

func (rs *RecommendationService) RecordSeenGame(userId, gameId string) error {
	seenGame := models.UserSeenGame{
		UserID: userId,
		GameID: gameId,
		SeenAt: time.Now(),
	}

	return rs.db.Exec(`
		INSERT INTO user_seen_games (user_id, game_id, seen_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, game_id)
		DO UPDATE SET seen_at = EXCLUDED.seen_at
	`, userId, gameId, seenGame.SeenAt).Error
}

type PlayInteraction struct {
	GameID   string
	PlayTime *int
}

func (rs *RecommendationService) RecordInteraction(userId string, interaction interface{}) error {
	var userInteraction models.UserGameInteraction
	var gameId string
	var interactionType string

	switch v := interaction.(type) {
	case PlayInteraction:
		gameId = v.GameID
		interactionType = "play"
		result := rs.db.Where("user_id = ? AND game_id = ?", userId, gameId).First(&userInteraction)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				userInteraction = models.UserGameInteraction{
					UserID: userId,
					GameID: gameId,
				}
			} else {
				return result.Error
			}
		}
		userInteraction.PlayCount++
		userInteraction.PlayTime += *v.PlayTime
	case string:
		gameId = v
		interactionType = "like"
		result := rs.db.Where("user_id = ? AND game_id = ?", userId, gameId).First(&userInteraction)
		if result.Error != nil {
			if result.Error == gorm.ErrRecordNotFound {
				userInteraction = models.UserGameInteraction{
					UserID: userId,
					GameID: gameId,
				}
			} else {
				return result.Error
			}
		}
		userInteraction.LikeCount++
	default:
		return fmt.Errorf("invalid interaction type")
	}

	userInteraction.LastInteraction = time.Now()

	if userInteraction.ID == 0 {
		if err := rs.db.Create(&userInteraction).Error; err != nil {
			return err
		}
	} else {
		if err := rs.db.Save(&userInteraction).Error; err != nil {
			return err
		}
	}

	var game models.Game
	if err := rs.db.First(&game, "id = ?", gameId).Error; err != nil {
		return err
	}

	switch interactionType {
	case "play":
		if err := rs.db.Model(&game).Updates(map[string]interface{}{
			"play_count": gorm.Expr("play_count + ?", 1),
			"play_time":  gorm.Expr("play_time + ?", userInteraction.PlayTime),
		}).Error; err != nil {
			return err
		}
	case "like":
		if err := rs.db.Model(&game).Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return err
		}
	}

	shouldInvalidate, err := rs.shouldInvalidateCache(userId)
	if err != nil {
		return err
	}

	if shouldInvalidate {
		cacheKey := fmt.Sprintf(recommendationCacheKey, userId)
		rs.redisClient.Del(context.Background(), cacheKey)
	}

	return nil
}

func (rs *RecommendationService) shouldInvalidateCache(userId string) (bool, error) {
	var interactionCount int64
	err := rs.db.Model(&models.UserGameInteraction{}).
		Where("user_id = ? AND last_interaction > ?", userId, time.Now().Add(-1*cacheExpirationTime)).
		Count(&interactionCount).Error
	if err != nil {
		return false, err
	}

	return interactionCount >= 5, nil
}
