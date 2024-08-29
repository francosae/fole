package handlers

import (
	"github.com/PixelzOrg/PHOLE.git/pkg/api/types"
	"github.com/PixelzOrg/PHOLE.git/pkg/services"
	"github.com/PixelzOrg/PHOLE.git/pkg/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type CommentHandler struct {
	service *services.CommentService
}

func NewCommentHandler(service *services.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

// CreateCommentByGameId godoc
// @Summary Create a new comment or reply
// @Description Create a new comment for a game or reply to an existing comment
// @Tags comments
// @Accept json
// @Produce json
// @Param gameId path string true "Game ID"
// @Param comment body types.CreateCommentRequest true "Comment Content and Optional Parent ID"
// @Success 200 {object} types.CommentResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /games/{gameId}/comments/create [post]
func (cs *CommentHandler) CreateCommentByGameId(c *gin.Context) {
	gameId := c.Param("gameId")
	userId := c.GetString("userId")

	var req types.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		return
	}

	if req.ParentID != nil {
		parentExists, err := cs.service.ValidateParentComment(*req.ParentID, gameId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Error validating parent comment"})
			return
		}
		if !parentExists {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid parent comment"})
			return
		}
	}

	if utils.ContainsNegativeWords(req.Content) {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Comment contains negative words."})
		return
	}

	comment, err := cs.service.CreateCommentByGameId(gameId, userId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	res := types.CommentResponse{
		ID:        comment.ID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
		UserID:    comment.UserID,
		GameID:    comment.GameID,
		ParentID:  comment.ParentID,
	}
	c.JSON(http.StatusOK, res)
}

// GetCommentsByGameId godoc
// @Summary Get comments for a game
// @Description Get paginated comments for a specific game
// @Tags comments
// @Accept json
// @Produce json
// @Param request body types.GetCommentsByGameIdRequest true "Pagination and Game ID"
// @Param gameId path string true "Game ID"
// @Success 200 {object} types.PaginatedResponse
// @Failure 400 {object} types.ErrorResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /games/{gameId}/comments/get [post]
func (cs *CommentHandler) GetCommentsByGameId(c *gin.Context) {
	var req types.GetCommentsByGameIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "Invalid request parameters: " + err.Error()})
		return
	}

	comments, totalItems, err := cs.service.GetCommentsByGameId(req.GameID, req.Pagination.Page, req.Pagination.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
		return
	}

	totalPages := (int(totalItems) + req.Pagination.PageSize - 1) / req.Pagination.PageSize

	res := types.PaginatedResponse{
		Data: types.GetCommentsByGameIdResponse{
			Comments: comments,
		},
		TotalItems: totalItems,
		TotalPages: totalPages,
		Page:       req.Pagination.Page,
		PageSize:   req.Pagination.PageSize,
	}

	c.JSON(http.StatusOK, res)
}

// DeleteCommentByCommentId godoc
// @Summary Delete a comment and its replies
// @Description Soft delete a comment and all its replies. Only the comment author or game creator can delete.
// @Tags comments
// @Accept json
// @Produce json
// @Param commentId path string true "Comment ID"
// @Param gameId path string true "Game ID"
// @Success 200 {object} types.SuccessResponse
// @Router /games/{gameId}/comments/{commentId} [delete]
func (cs *CommentHandler) DeleteCommentByCommentId(c *gin.Context) {
	commentId := c.Param("commentId")
	userId := c.Param("userId")

	err := cs.service.DeleteCommentByCommentId(commentId, userId)
	if err != nil {
		switch err.Error() {
		case types.NotFoundMessage:
			c.JSON(http.StatusNotFound, types.ErrorResponse{Error: types.NotFoundMessage})
		case types.UnauthorizedMessage:
			c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: types.UnauthorizedMessage})
		default:
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: types.FailedMessage + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse{Status: types.SuccessMessage})
}
