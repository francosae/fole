package middleware

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/supabase"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
)

func AuthMiddleware(supabaseAuth *supabase.SupabaseAuth) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		bearerToken := strings.SplitN(authHeader, " ", 2)
		if len(bearerToken) != 2 || !strings.EqualFold(bearerToken[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer token"})
			return
		}

		token := bearerToken[1]

		user, err := supabaseAuth.VerifyToken(token)
		if err != nil {
			log.Warn().Err(err).Str("token", token).Msg("Error verifying Supabase token")
			handleAuthError(c, err)
			return
		}

		setUserContext(c, user)
		log.Info().Str("userId", user.ID).Str("userEmail", user.Email).Msg("User authenticated successfully")

		c.Next()
	}
}

func setUserContext(c *gin.Context, user *supabase.User) {
	c.Set("userId", user.ID)
	c.Set("userEmail", user.Email)
}

func handleAuthError(c *gin.Context, err error) {
	switch err.Error() {
	case "token has expired":
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has expired"})
	case "invalid token":
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
	default:
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
	}
}
