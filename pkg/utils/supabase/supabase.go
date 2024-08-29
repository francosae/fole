package supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
)

const (
	jwksCacheKey = "supabase:jwks"
	jwksTTL      = 24 * time.Hour
)

type SupabaseAuth struct {
	ProjectURL  string
	APIKey      string
	RedisClient *redis.Client
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewSupabaseAuth(projectURL, apiKey string, redisClient *redis.Client) *SupabaseAuth {
	return &SupabaseAuth{
		ProjectURL:  projectURL,
		APIKey:      apiKey,
		RedisClient: redisClient,
	}
}

func (sa *SupabaseAuth) VerifyToken(tokenString string) (*User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		jwks, err := sa.getJWKS()
		if err != nil {
			return nil, err
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid not found in token header")
		}

		key, ok := jwks[kid]
		if !ok {
			return nil, errors.New("key not found in JWKS")
		}

		return key, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("expiration time not found in token")
	}
	if time.Now().Unix() > int64(exp) {
		return nil, errors.New("token has expired")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("user ID not found in token")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, errors.New("email not found in token")
	}

	return &User{
		ID:    userID,
		Email: email,
	}, nil
}

func (sa *SupabaseAuth) getJWKS() (map[string]interface{}, error) {
	ctx := context.Background()

	cachedJWKS, err := sa.RedisClient.Get(ctx, jwksCacheKey).Result()
	if err == nil {
		var jwks map[string]interface{}
		err = json.Unmarshal([]byte(cachedJWKS), &jwks)
		if err == nil {
			return jwks, nil
		}
	}

	jwks, err := sa.fetchJWKS()
	if err != nil {
		return nil, err
	}

	jwksJSON, err := json.Marshal(jwks)
	if err == nil {
		sa.RedisClient.Set(ctx, jwksCacheKey, jwksJSON, jwksTTL)
	}

	return jwks, nil
}

func (sa *SupabaseAuth) fetchJWKS() (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/auth/v1/jwks", sa.ProjectURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	keys := make(map[string]interface{})
	for _, key := range jwks.Keys {
		kid, ok := key["kid"].(string)
		if !ok {
			continue
		}
		keys[kid] = key
	}

	return keys, nil
}

func (sa *SupabaseAuth) GetUser(userId string) (*User, error) {
	url := fmt.Sprintf("%s/auth/v1/admin/users/%s", sa.ProjectURL, userId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("apikey", sa.APIKey)
	req.Header.Set("Authorization", "Bearer "+sa.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user not found or unauthorized access")
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}
