package services

import (
	"errors"
	"fmt"
	"github.com/PixelzOrg/PHOLE.git/pkg/api/types"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils/supabase"
	"gorm.io/gorm"
	"time"
)

type UserService struct {
	databaseHandler database.Handler
	supabaseAuth    *supabase.SupabaseAuth
}

func NewUserService(databaseHandler database.Handler, supabaseAuth *supabase.SupabaseAuth) *UserService {
	return &UserService{
		databaseHandler: databaseHandler,
		supabaseAuth:    supabaseAuth,
	}
}

func (us *UserService) CreateUser(userId string, newUser types.CreateUserRequest) (*models.User, error) {
	_, err := us.supabaseAuth.GetUser(userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from Supabase: %w", err)
	}

	var existingUser models.User
	if err := us.databaseHandler.DB.Where("uid = ?", userId).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("user already exists in the database")
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("error checking for existing user: %w", err)
	}

	user := models.User{
		UID:             userId,
		Email:           newUser.Email,
		Username:        newUser.Username,
		DisplayName:     &newUser.DisplayName,
		ProfileImageURL: newUser.ProfileImageURL,
		Bio:             newUser.Bio,
		Gender:          newUser.Gender,
		Birthday:        newUser.Birthday,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := us.databaseHandler.DB.Create(&newUser).Error; err != nil {
		return nil, fmt.Errorf("failed to create user in database: %w", err)
	}

	return &user, nil
}

func (us *UserService) GetUserProfileById(userId string) (*types.GetUserProfileResponse, error) {
	var user models.User
	if err := us.databaseHandler.DB.Where("uid = ?", userId).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user from database: %w", err)
	}

	return &types.GetUserProfileResponse{
		Username:        user.Username,
		DisplayName:     user.DisplayName,
		ProfileImageURL: user.ProfileImageURL,
		Bio:             user.Bio,
		FollowersCount:  user.FollowersCount,
		FollowingCount:  user.FollowingCount,
	}, nil
}

func (us *UserService) UpdateUserProfileById(userId string, req types.UpdateUserProfileRequest) (*types.UpdateUserProfileResponse, error) {
	var user models.User
	if err := us.databaseHandler.DB.Where("uid = ?", userId).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("User not found")
		}
		return nil, fmt.Errorf("Failed to get user from database: %w", err)
	}

	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}
	if req.ProfileImageURL != nil {
		user.ProfileImageURL = req.ProfileImageURL
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}
	if req.Gender != nil {
		user.Gender = req.Gender
	}

	if err := us.databaseHandler.DB.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	return &types.UpdateUserProfileResponse{Status: "success"}, nil
}

func (us *UserService) GetGamesCreatedByUserId(userId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	var games []models.Game
	var totalItems int64

	if err := us.databaseHandler.DB.Model(&models.Game{}).Where("creator_id = ?", userId).Count(&totalItems).Error; err != nil {
		return nil, fmt.Errorf("Failed to count games: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := us.databaseHandler.DB.Where("creator_id = ?", userId).
		Order("created_at DESC").
		Offset(offset).
		Limit(pagination.PageSize).
		Find(&games).Error; err != nil {
		return nil, fmt.Errorf("Failed to get games from database: %w", err)
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       games,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}

func (us *UserService) CreateFollow(followerId, followedId string) error {
	return us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		var follower, followed models.User
		if err := tx.First(&follower, "uid = ?", followerId).Error; err != nil {
			return errors.New(types.NotFoundMessage)
		}
		if err := tx.First(&followed, "uid = ?", followedId).Error; err != nil {
			return errors.New(types.NotFoundMessage)
		}
		if followerId == followedId {
			return errors.New(types.UnauthorizedMessage)
		}
		var existingFollow models.Follow
		err := tx.Where("follower_id = ? AND following_id = ?", followerId, followedId).First(&existingFollow).Error
		if err == nil {
			return errors.New(types.FailedMessage)
		} else if err != gorm.ErrRecordNotFound {
			return errors.New(types.FailedMessage)
		}
		follow := models.Follow{
			FollowerID:  followerId,
			FollowingID: followedId,
		}
		if err := tx.Create(&follow).Error; err != nil {
			return errors.New(types.FailedMessage)
		}

		if err := tx.Model(&follower).Update("following_count", gorm.Expr("following_count + ?", 1)).Error; err != nil {
			return errors.New(types.FailedMessage)
		}
		if err := tx.Model(&followed).Update("followers_count", gorm.Expr("followers_count + ?", 1)).Error; err != nil {
			return errors.New(types.FailedMessage)
		}

		return nil
	})
}

func (us *UserService) DeleteFollow(followerId, followedId string) error {
	return us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		var follower, followed models.User
		if err := tx.First(&follower, "uid = ?", followerId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: follower not found", types.ErrNotFound)
			}
			return fmt.Errorf("error fetching follower: %w", err)
		}
		if err := tx.First(&followed, "uid = ?", followedId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: followed user not found", types.ErrNotFound)
			}
			return fmt.Errorf("error fetching followed user: %w", err)
		}

		var follow models.Follow
		if err := tx.Where("follower_id = ? AND following_id = ?", followerId, followedId).First(&follow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: follow relationship does not exist", types.ErrNotFound)
			}
			return fmt.Errorf("error checking follow relationship: %w", err)
		}

		if err := tx.Delete(&follow).Error; err != nil {
			return fmt.Errorf("failed to delete follow: %w", err)
		}

		if err := tx.Model(&follower).Update("following_count", gorm.Expr("GREATEST(following_count - 1, 0)")).Error; err != nil {
			return fmt.Errorf("failed to update follower's following count: %w", err)
		}
		if err := tx.Model(&followed).Update("followers_count", gorm.Expr("GREATEST(followers_count - 1, 0)")).Error; err != nil {
			return fmt.Errorf("failed to update followed user's followers count: %w", err)
		}

		return nil
	})
}

func (us *UserService) GetFollowersByUserId(userId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	var followers []models.User
	var totalItems int64

	err := us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "uid = ?", userId).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("user not found")
			}
			return err
		}
		if err := tx.Model(&models.Follow{}).Where("following_id = ?", userId).Count(&totalItems).Error; err != nil {
			return err
		}

		offset := (pagination.Page - 1) * pagination.PageSize
		err := tx.Table("users").
			Joins("JOIN follows ON users.uid = follows.follower_id").
			Where("follows.following_id = ?", userId).
			Offset(offset).
			Limit(pagination.PageSize).
			Find(&followers).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       followers,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}

func (us *UserService) GetFollowingByUserId(userId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	var following []models.User
	var totalItems int64

	err := us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "uid = ?", userId).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("user not found")
			}
			return err
		}

		if err := tx.Model(&models.Follow{}).Where("follower_id = ?", userId).Count(&totalItems).Error; err != nil {
			return err
		}

		offset := (pagination.Page - 1) * pagination.PageSize
		err := tx.Table("users").
			Joins("JOIN follows ON users.uid = follows.following_id").
			Where("follows.follower_id = ?", userId).
			Offset(offset).
			Limit(pagination.PageSize).
			Find(&following).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       following,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}

func (us *UserService) GetLikedGamesByUserId(userId string, requesterId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	if userId != requesterId {
		return nil, types.ErrUnauthorized
	}

	var likedGames []models.Game
	var totalItems int64

	err := us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&models.User{}, "uid = ?", userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found")
			}
			return err
		}

		if err := tx.Model(&models.Like{}).Where("user_id = ?", userId).Count(&totalItems).Error; err != nil {
			return err
		}

		offset := (pagination.Page - 1) * pagination.PageSize
		err := tx.Table("games").
			Joins("JOIN likes ON games.id = likes.game_id").
			Where("likes.user_id = ?", userId).
			Offset(offset).
			Limit(pagination.PageSize).
			Find(&likedGames).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       likedGames,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}

func (us *UserService) GetBookmarkedGamesByUserId(userId string, requesterId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	if userId != requesterId {
		return nil, types.ErrUnauthorized
	}

	var bookmarkedGames []models.Game
	var totalItems int64

	err := us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&models.User{}, "uid = ?", userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found")
			}
			return err
		}

		if err := tx.Model(&models.Bookmark{}).Where("user_id = ?", userId).Count(&totalItems).Error; err != nil {
			return err
		}

		offset := (pagination.Page - 1) * pagination.PageSize
		err := tx.Table("games").
			Joins("JOIN bookmarks ON games.id = bookmarks.game_id").
			Where("bookmarks.user_id = ?", userId).
			Offset(offset).
			Limit(pagination.PageSize).
			Find(&bookmarkedGames).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       bookmarkedGames,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}

func (us *UserService) GetRecentlyPlayedGamesByUserId(userId string, requesterId string, pagination types.PaginationQuery) (*types.PaginatedResponse, error) {
	if userId != requesterId {
		return nil, types.ErrUnauthorized
	}

	var recentlyPlayedGames []models.Game
	var totalItems int64

	err := us.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&models.User{}, "uid = ?", userId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found")
			}
			return err
		}

		if err := tx.Model(&models.RecentlyPlayed{}).Where("user_id = ?", userId).Count(&totalItems).Error; err != nil {
			return err
		}

		offset := (pagination.Page - 1) * pagination.PageSize
		err := tx.Table("games").
			Joins("JOIN recently_played ON games.id = recently_played.game_id").
			Where("recently_played.user_id = ?", userId).
			Order("recently_played.played_at DESC").
			Offset(offset).
			Limit(pagination.PageSize).
			Find(&recentlyPlayedGames).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	totalPages := (int(totalItems) + pagination.PageSize - 1) / pagination.PageSize

	return &types.PaginatedResponse{
		Data:       recentlyPlayedGames,
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
	}, nil
}
