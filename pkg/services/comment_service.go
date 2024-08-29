package services

import (
	"errors"
	"github.com/PixelzOrg/PHOLE.git/pkg/api/types"
	"github.com/PixelzOrg/PHOLE.git/pkg/database"
	"github.com/PixelzOrg/PHOLE.git/pkg/models"
	"gorm.io/gorm"
)

type CommentService struct {
	databaseHandler database.Handler
}

func NewCommentService(databaseHandler database.Handler) *CommentService {
	return &CommentService{
		databaseHandler: databaseHandler,
	}
}

func (cs *CommentService) ValidateParentComment(parentID, gameID string) (bool, error) {
	var count int64
	err := cs.databaseHandler.DB.Model(&models.Comment{}).
		Where("id = ? AND game_id = ? AND parent_id IS NULL", parentID, gameID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (cs *CommentService) CreateCommentByGameId(gameId, userId string, req types.CreateCommentRequest) (comment models.Comment, err error) {
	tx := cs.databaseHandler.DB.Begin()

	comment = models.Comment{
		Content:  req.Content,
		UserID:   userId,
		GameID:   gameId,
		ParentID: req.ParentID,
	}

	if err = tx.Create(&comment).Error; err != nil {
		tx.Rollback()
		return
	}

	if err = tx.Model(&models.Game{}).Where("id = ?", gameId).UpdateColumn("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
		tx.Rollback()
		return
	}

	if err = tx.Commit().Error; err != nil {
		return
	}

	return
}

func (cs *CommentService) GetCommentsByGameId(gameId string, page, pageSize int) ([]types.CommentResponse, int64, error) {
	var comments []models.Comment
	var totalItems int64
	var responseComments []types.CommentResponse

	offset := (page - 1) * pageSize

	if err := cs.databaseHandler.DB.Model(&models.Comment{}).Where("game_id = ? AND parent_id IS NULL AND is_deleted = false", gameId).Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	if err := cs.databaseHandler.DB.Where("game_id = ? AND parent_id IS NULL AND is_deleted = false", gameId).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Where("is_deleted = false").Order("created_at ASC")
		}).
		Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	for _, comment := range comments {
		responseComment := types.CommentResponse{
			ID:        comment.ID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			UserID:    comment.UserID,
			GameID:    comment.GameID,
		}

		for _, reply := range comment.Replies {
			responseReply := types.CommentResponse{
				ID:        reply.ID,
				Content:   reply.Content,
				CreatedAt: reply.CreatedAt,
				UserID:    reply.UserID,
				GameID:    reply.GameID,
				ParentID:  &comment.ID,
			}
			responseComment.Replies = append(responseComment.Replies, responseReply)
		}

		responseComments = append(responseComments, responseComment)
	}

	return responseComments, totalItems, nil
}

func (cs *CommentService) DeleteCommentByCommentId(commentId, userId string) error {
	var comment models.Comment
	if err := cs.databaseHandler.DB.Preload("Game").First(&comment, "id = ?", commentId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(types.NotFoundMessage)
		}
		return err
	}

	if comment.UserID != userId && *comment.Game.CreatorID != userId {
		return errors.New(types.UnauthorizedMessage)
	}

	return cs.databaseHandler.DB.Transaction(func(tx *gorm.DB) error {
		var replyCount int64
		if err := tx.Model(&models.Comment{}).Where("parent_id = ? AND is_deleted = false", commentId).Count(&replyCount).Error; err != nil {
			return err
		}

		if err := tx.Model(&comment).Update("is_deleted", true).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Comment{}).Where("parent_id = ?", commentId).Update("is_deleted", true).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Game{}).Where("id = ?", comment.GameID).
			UpdateColumn("comment_count", gorm.Expr("GREATEST(comment_count - ?, 0)", replyCount+1)).Error; err != nil {
			return err
		}

		return nil
	})
}
