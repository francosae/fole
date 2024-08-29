package main

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/api"
	"github.com/PixelzOrg/PHOLE.git/pkg/config"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/middleware"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/firebase"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/supabase"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"net/http"
	"os"
	"time"
)

// @title Hitbox Backend AKA The P-HOLE
// @version 1.0
// @description All you could need
// COMMENT @host http://hitbox-platform-api-env.eba-fa2m2mxn.us-east-1.elasticbeanstalk.com
// @host localhost:8080
// @BasePath /api/v1
func main() {
	c, err := config.LoadConfig()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed at loading config")
	}

	h := database.Init(c.ConnectionString)

	opt, _ := redis.ParseURL(c.RedisCredentialsPath)
	redisClient := redis.NewClient(opt)

	fa, err := firebase.NewFirebaseApp(c.FirebaseCredentialsPath, c.BackupFirebasePath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Firebase")
	}
	defer fa.Close()

	supabaseAuth := supabase.NewSupabaseAuth(c.SupabaseProjectURL, c.SupabaseAPIKey, redisClient)
	if supabaseAuth == nil {
		log.Fatal().Msg("Failed to initialize Supabase")
	}

	algoliaClient := search.NewClient(c.AlgoliaAppId, c.AlgoliaKey)
	if algoliaClient == nil {
		log.Fatal().Msg("Failed to initialize Algolia")
	}

	//ctx := context.Background()
	//log.Printf("Starting migration")
	//err = utils.MigrateFromFirestore(ctx, fa, h, algoliaClient, "hitbox-games-bucket")
	//err = utils.MigrateFromCSV(ctx, h, algoliaClient, "hitbox-games-bucket", "./games.csv")
	//if err != nil {
	//	log.Fatal().Err(err).Msg("Failed to migrate from Firestore")
	//}
	//log.Print("Migration completed successfully")

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Logger()

	r := gin.Default()
	r.Use(middleware.LogMiddleware())

	//swagger shit
	r.Static("/docs", "../docs")
	swaggerURL := ginSwagger.URL("/docs/swagger.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, swaggerURL))
	//health check
	r.GET("/health-check", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})
	api.SetupRoutes(r, h, supabaseAuth, redisClient, algoliaClient)

	log.Info().Msg("🚀🚀🚀 Hitbox P-HOLE is running 🚀🚀🚀")
	if err := r.Run(c.Port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start up the P-HOLE")
	}
}
