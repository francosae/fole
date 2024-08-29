package handlers

import (
	"errors"
	"github.com/PixelzOrg/PHOLE.git/pkg/api/types"
	"github.com/PixelzOrg/PHOLE.git/pkg/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user in the system after Supabase authentication
// @Tags users
// @Accept json
// @Produce json
// @Param Authorization header string true "UID token"
// @Param user body types.CreateUserRequest true "User information"
// @Success 201 {object} types.CreateUserResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/createUser [post]
func (uh *UserHandler) CreateUser(c *gin.Context) {
	userID := c.GetString("userId")

	if userID == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{"User ID is required"})
		return
	}

	var newUser types.CreateUserRequest

	if err := c.BindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{"Invalid request payload"})
		return
	}

	user, err := uh.service.CreateUser(userID, newUser)

	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{err.Error()})
		return
	}

	c.JSON(http.StatusCreated, types.CreateUserResponse{User: user})
}

// GetUserProfileById godoc
// @Summary Get user profile
// @Description Get profile for user, following and followers count
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Success 200 {object} types.GetUserProfileResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/profile/{userId} [get]
func (uh *UserHandler) GetUserProfileById(c *gin.Context) {
	userId := c.GetString("userId")

	if userId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{"User ID is required"})
		return
	}

	user, err := uh.service.GetUserProfileById(userId)

	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{"Failed to get user profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUserProfileById godoc
// @Summary Update user profile
// @Description Update the profile information for the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param request body types.UpdateUserProfileRequest true "Update User Profile Request"
// @Success 200 {object} types.UpdateUserProfileResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/profile/{userId} [PATCH]
func (uh *UserHandler) UpdateUserProfileById(c *gin.Context) {
	userId := c.GetString("userId")
	paramUserId := c.Param("userId")

	if userId == "" {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "User not authenticated"})
		return
	}

	if userId != paramUserId {
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "Not authorized to update this profile"})
		return
	}

	var req types.UpdateUserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid request payload: " + err.Error()})
		return
	}

	updatedProfile, err := uh.service.UpdateUserProfileById(userId, req)
	if err != nil {
		switch err.Error() {
		case "user not found":
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to update user profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, updatedProfile)
}

// GetGamesCreatedByUserId godoc
// @Summary Get games created by user
// @Description Get paginated list of games created by a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/games [get]
func (uh *UserHandler) GetGamesCreatedByUserId(c *gin.Context) {
	userId := c.Param("userId")

	if userId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "User ID is required"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	paginatedGames, err := uh.service.GetGamesCreatedByUserId(userId, pagination)

	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get games created by user"})
		return
	}

	c.JSON(http.StatusOK, paginatedGames)
}

// CreateFollow godoc
// @Summary Follow a user
// @Description Create a new follow relationship between the authenticated user and the target user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID to follow"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/follows [post]
func (uh *UserHandler) CreateFollow(c *gin.Context) {
	followerId := c.GetString("userId")
	followedId := c.Param("userId")

	if followerId == "" {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "User not authenticated"})
		return
	}

	if followedId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "User ID to follow is required"})
		return
	}

	err := uh.service.CreateFollow(followerId, followedId)

	if err != nil {
		switch err.Error() {
		case types.NotFoundMessage:
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: err.Error()})
		case types.UnauthorizedMessage:
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		case types.FailedMessage:
			c.JSON(http.StatusConflict, types.ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to follow user: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse{Status: "Successfully followed user"})
}

// DeleteFollow godoc
// @Summary Unfollow a user
// @Description Remove a follow relationship between the authenticated user and the target user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID to unfollow"
// @Success 200 {object} types.SuccessResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/follows [delete]
func (uh *UserHandler) DeleteFollow(c *gin.Context) {
	followerId := c.GetString("userId")
	followedId := c.Param("userId")

	if followerId == "" {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "User not authenticated"})
		return
	}

	if followedId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "User ID to unfollow is required"})
		return
	}

	err := uh.service.DeleteFollow(followerId, followedId)

	if err != nil {
		switch {
		case errors.Is(err, types.ErrNotFound):
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: err.Error()})
		case errors.Is(err, types.ErrUnauthorized):
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to unfollow user: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse{Status: "Successfully unfollowed user"})
}

// GetFollowersByUserId godoc
// @Summary Get followers of a user
// @Description Get paginated list of followers for a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/followers [get]
func (uh *UserHandler) GetFollowersByUserId(c *gin.Context) {
	userId := c.Param("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "User ID is required"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	followers, err := uh.service.GetFollowersByUserId(userId, pagination)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get followers"})
		}
		return
	}

	c.JSON(http.StatusOK, followers)
}

// GetFollowingByUserId godoc
// @Summary Get users followed by a user
// @Description Get paginated list of users followed by a specific user
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/following [get]
func (uh *UserHandler) GetFollowingByUserId(c *gin.Context) {
	userId := c.Param("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "User ID is required"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	following, err := uh.service.GetFollowingByUserId(userId, pagination)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get following users"})
		}
		return
	}

	c.JSON(http.StatusOK, following)
}

// GetLikedGamesByUserId godoc
// @Summary Get liked games of a user
// @Description Get paginated list of games liked by a specific user (only accessible by the user themselves)
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/likedGames [get]
func (uh *UserHandler) GetLikedGamesByUserId(c *gin.Context) {
	userId := c.Param("userId")
	requesterId := c.GetString("userId")

	if userId == "" || requesterId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	likedGames, err := uh.service.GetLikedGamesByUserId(userId, requesterId, pagination)
	if err != nil {
		switch err {
		case types.ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "Unauthorized access"})
		case gorm.ErrRecordNotFound:
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get liked games"})
		}
		return
	}

	c.JSON(http.StatusOK, likedGames)
}

// GetBookmarkedGamesByUserId godoc
// @Summary Get bookmarked games of a user
// @Description Get paginated list of games bookmarked by a specific user (only accessible by the user themselves)
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/bookmarkedGames [get]
func (uh *UserHandler) GetBookmarkedGamesByUserId(c *gin.Context) {
	userId := c.Param("userId")
	requesterId := c.GetString("userId")

	if userId == "" || requesterId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	bookmarkedGames, err := uh.service.GetBookmarkedGamesByUserId(userId, requesterId, pagination)
	if err != nil {
		switch err {
		case types.ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "Unauthorized access"})
		case gorm.ErrRecordNotFound:
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get bookmarked games"})
		}
		return
	}

	c.JSON(http.StatusOK, bookmarkedGames)
}

// GetRecentlyPlayedGamesByUserId godoc
// @Summary Get recently played games of a user
// @Description Get paginated list of games recently played by a specific user (only accessible by the user themselves)
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /users/{userId}/recentlyPlayedGames [get]
func (uh *UserHandler) GetRecentlyPlayedGamesByUserId(c *gin.Context) {
	userId := c.Param("userId")
	requesterId := c.GetString("userId")

	if userId == "" || requesterId == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid user ID"})
		return
	}

	var pagination types.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid pagination parameters"})
		return
	}

	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	recentlyPlayedGames, err := uh.service.GetRecentlyPlayedGamesByUserId(userId, requesterId, pagination)
	if err != nil {
		switch err {
		case types.ErrUnauthorized:
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "Unauthorized access"})
		case gorm.ErrRecordNotFound:
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "User not found"})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get recently played games"})
		}
		return
	}

	c.JSON(http.StatusOK, recentlyPlayedGames)
}
