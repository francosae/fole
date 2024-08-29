package api

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/api/handlers"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/middleware"
	"github.com/PixelzOrg/PHOLE.git/pkg/services"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/supabase"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRoutes(r *gin.Engine, databaseHandler database.Handler, supabaseAuth *supabase.SupabaseAuth, redisClient *redis.Client, algoliaClient *search.Client) {
	recommendationService := services.NewRecommendationService(databaseHandler, redisClient)
	gameService := services.NewGameService(databaseHandler, supabaseAuth, algoliaClient)
	gameHandler := handlers.NewGameHandler(gameService, recommendationService)

	commentService := services.NewCommentService(databaseHandler)
	commentHandler := handlers.NewCommentHandler(commentService)
	userService := services.NewUserService(databaseHandler, supabaseAuth)
	userHandler := handlers.NewUserHandler(userService)

	// TODO: Use keyset pagination for everything
	v1 := r.Group("/api/v1")
	{
		games := v1.Group("/games")
		{
			games.GET("/feed", gameHandler.Feed)
			games.GET("/:gameId", gameHandler.GameDetailsByGameId)

			// Like, Bookmark, Play/View
			games.POST("/:gameId/interactions", middleware.AuthMiddleware(supabaseAuth), gameHandler.CreateInteractionByGameId)

			// Comments
			comments := games.Group("/:gameId/comments")
			{
				comments.POST("/create", middleware.AuthMiddleware(supabaseAuth), commentHandler.CreateCommentByGameId)
				comments.POST("/get", commentHandler.GetCommentsByGameId)
				comments.DELETE("/:commentId", middleware.AuthMiddleware(supabaseAuth), commentHandler.DeleteCommentByCommentId)
			}
		}

		users := v1.Group("/users")
		{
			// TODO: USER PROFILE PICTURES!?!?!?!
			users.POST("/createUser", userHandler.CreateUser)
			users.GET("/profile/:userId", userHandler.GetUserProfileById)
			users.PATCH("/profile/:userId", middleware.AuthMiddleware(supabaseAuth), userHandler.UpdateUserProfileById)
			users.GET("/:userId/games", userHandler.GetGamesCreatedByUserId)
			//  LIKED, BOOKMARKED, RECENTLY PLAYED GAMES BY USER
			users.GET("/:userId/likedGames", middleware.AuthMiddleware(supabaseAuth), userHandler.GetLikedGamesByUserId)
			users.GET("/:userId/bookmarkedGames", middleware.AuthMiddleware(supabaseAuth), userHandler.GetBookmarkedGamesByUserId)
			users.GET("/:userId/recentlyPlayedGames", middleware.AuthMiddleware(supabaseAuth), userHandler.GetRecentlyPlayedGamesByUserId)

			// Following and Followers
			users.POST("/:userId/follows", middleware.AuthMiddleware(supabaseAuth), userHandler.CreateFollow)
			users.DELETE("/:userId/follows/:followId", middleware.AuthMiddleware(supabaseAuth), userHandler.DeleteFollow)
			users.GET("/:userId/followers", userHandler.GetFollowersByUserId)
			users.GET("/:userId/following", userHandler.GetFollowingByUserId)
		}
	}
}
