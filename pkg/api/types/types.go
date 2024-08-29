package types

import (
	"errors"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"time"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Status string `json:"status"`
}

// --- Games ---
type FeedResponse struct {
	Games      []models.Game `json:"games"`
	TotalGames int64         `json:"totalGames"`
	Page       int           `json:"page"`
	Limit      int           `json:"limit"`
}

type GameDetailsResponse struct {
	Game models.Game `json:"game"`
}

type CreateInteractionRequest struct {
	Type     string `json:"type" binding:"required,oneof=play like bookmark"`
	PlayTime *int   `json:"playtime,omitempty"`
}

type CreateInteractionResponse struct {
	Status string `json:"status"`
}

type RecordSeenGameResponse struct {
	Status string `json:"status"`
}

// --- Comments ---
// TODO: Update the types below to use errors.Is() instead of string comparison
const (
	NotFoundMessage     = "Not Found"
	UnauthorizedMessage = "Unauthorized for operation"
	FailedMessage       = "Failed operation"
	SuccessMessage      = "Successful operation"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type PaginationQuery struct {
	Page     int `json:"page" binding:"required,min=1"`
	PageSize int `json:"page_size" binding:"required,min=1"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
}
type CommentResponse struct {
	ID        string            `json:"id"`
	Content   string            `json:"content"`
	CreatedAt time.Time         `json:"created_at"`
	UserID    string            `json:"user_id"`
	GameID    string            `json:"game_id"`
	ParentID  *string           `json:"parent_id,omitempty"`
	Replies   []CommentResponse `json:"replies,omitempty"`
}

type CreateCommentRequest struct {
	Content  string  `json:"content" binding:"required"`
	ParentID *string `json:"parent_id,omitempty"`
}

type CreateCommentResponse struct {
	NewComment models.Comment `json:"comment"`
	Status     string         `json:"status"`
}

type GetCommentsByGameIdRequest struct {
	GameID     string          `json:"game_id" binding:"required"`
	Pagination PaginationQuery `json:"pagination"`
}

type GetCommentsByGameIdResponse struct {
	Comments []CommentResponse `json:"comments"`
}

// --- Users ---
type CreateUserRequest struct {
	Email           string     `json:"email" binding:"required,email"`
	Username        string     `json:"username" binding:"required"`
	DisplayName     string     `json:"displayName"`
	ProfileImageURL *string    `json:"profileImageURL"`
	Bio             *string    `json:"bio"`
	Gender          *string    `json:"gender"`
	Birthday        *time.Time `json:"birthday"`
}

type CreateUserResponse struct {
	User *models.User `json:"user"`
}

type GetUserProfileResponse struct {
	Username        string `gorm:"unique"`
	DisplayName     *string
	ProfileImageURL *string
	Bio             *string
	FollowersCount  int `gorm:"default:0"`
	FollowingCount  int `gorm:"default:0"`
}

type UpdateUserProfileRequest struct {
	DisplayName     *string `json:"displayName"`
	ProfileImageURL *string `json:"profileImageURL"`
	Bio             *string `json:"bio"`
	Gender          *string `json:"gender"`
}

type UpdateUserProfileResponse struct {
	Status string `json:"status"`
}
