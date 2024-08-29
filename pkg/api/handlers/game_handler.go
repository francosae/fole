package handlers

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/api/types"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"github.com/PixelzOrg/PHOLE.git/pkg/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

type GameHandler struct {
	gameService           *services.GameService
	recommendationService *services.RecommendationService
}

func NewGameHandler(gameService *services.GameService, recommendationService *services.RecommendationService) *GameHandler {
	return &GameHandler{
		gameService:           gameService,
		recommendationService: recommendationService,
	}
}

// Feed godoc
// @Summary Get a feed of recommended games
// @Description Get a paginated feed of recommended games for the user or fallback recommendations for anonymous users
// @Tags games
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Success 200 {object} types.FeedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /games/feed [get]
func (gh *GameHandler) Feed(c *gin.Context) {
	userId := c.GetString("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))

	var games []models.Game
	var totalGames int64
	var err error

	if userId != "" {
		games, totalGames, err = gh.recommendationService.GetRecommendations(userId, page, limit)
	} else {
		games, totalGames, err = gh.recommendationService.GetFallbackRecommendations(page, limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recommendations"})
		return
	}

	res := types.FeedResponse{
		Games:      games,
		TotalGames: totalGames,
		Page:       page,
		Limit:      limit,
	}

	c.JSON(http.StatusOK, res)
}

// GameDetailsByGameId godoc
// @Summary Get details of a game by game ID
// @Description Get details of a game by game ID
// @Tags games
// @Accept json
// @Produce json
// @Param gameId path string true "Game ID"
// @Success 200 {object} types.GameDetailsResponse
// @Router /games/{gameId} [get]
func (gh *GameHandler) GameDetailsByGameId(c *gin.Context) {
	gameId := c.Param("gameId")

	game, err := gh.gameService.GameDetailsByGameId(gameId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get game details"})
		return
	}

	res := types.GameDetailsResponse{
		Game: game,
	}

	c.JSON(http.StatusOK, res)
}

// CreateInteractionByGameId godoc
// @Summary Create an interaction for a game
// @Description Create an interaction (play, like, bookmark) for a game
// @Tags games
// @Accept json
// @Produce json
// @Param gameId path string true "Game ID"
// @Param request body types.CreateInteractionRequest true "Interaction details"
// @Success 200 {object} types.CreateInteractionResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /games/{gameId}/interactions [post]
func (gh *GameHandler) CreateInteractionByGameId(c *gin.Context) {
	gameId := c.Param("gameId")
	userId := c.GetString("userId") // Assume this is set by auth middleware

	var req types.CreateInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := gh.gameService.CreateInteractionByGameId(gameId, userId, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create interaction"})
		return
	}

	var interaction interface{}
	switch req.Type {
	case "play":
		interaction = services.PlayInteraction{
			GameID:   gameId,
			PlayTime: req.PlayTime,
		}
	case "like", "bookmark":
		interaction = gameId
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interaction type"})
		return
	}

	err = gh.recommendationService.RecordInteraction(userId, interaction)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "Interaction recorded, but failed to update recommendations"})
		return
	}

	res := types.CreateInteractionResponse{
		Status: "Interaction recorded successfully",
	}

	c.JSON(http.StatusOK, res)
}

// RecordSeenGame godoc
// @Summary Record a game as seen by the user
// @Description Record a game as seen by the user to improve recommendations
// @Tags games
// @Accept json
// @Produce json
// @Param gameId path string true "Game ID"
// @Success 200 {object} types.RecordSeenGameResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /games/{gameId}/seen [post]
func (gh *GameHandler) RecordSeenGame(c *gin.Context) {
	gameId := c.Param("gameId")
	userId := c.GetString("userId")

	err := gh.recommendationService.RecordSeenGame(userId, gameId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record seen game"})
		return
	}

	res := types.RecordSeenGameResponse{
		Status: "Game recorded as seen successfully",
	}

	c.JSON(http.StatusOK, res)
}
