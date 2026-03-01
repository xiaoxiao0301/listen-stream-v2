package handler

import (
	"errors"
	"net/http"

	"user-svc/internal/domain"

	"github.com/gin-gonic/gin"
)

// handleError 统一处理domain错误并返回适当的HTTP状态码
func handleError(c *gin.Context, err error) {
	switch {
	// 404 Not Found
	case errors.Is(err, domain.ErrFavoriteNotFound),
		errors.Is(err, domain.ErrHistoryNotFound),
		errors.Is(err, domain.ErrPlaylistNotFound),
		errors.Is(err, domain.ErrSongNotInPlaylist):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})

	// 409 Conflict
	case errors.Is(err, domain.ErrFavoriteAlreadyExists),
		errors.Is(err, domain.ErrPlaylistAlreadyExists),
		errors.Is(err, domain.ErrSongAlreadyInPlaylist):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})

	// 400 Bad Request
	case errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidSongID),
		errors.Is(err, domain.ErrInvalidSongName),
		errors.Is(err, domain.ErrInvalidDuration),
		errors.Is(err, domain.ErrInvalidPlaylistID),
		errors.Is(err, domain.ErrInvalidPlaylistName),
		errors.Is(err, domain.ErrPlaylistNameTooLong),
		errors.Is(err, domain.ErrPlaylistDescriptionTooLong),
		errors.Is(err, domain.ErrInvalidPosition):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

	// 403 Forbidden
	case errors.Is(err, domain.ErrUnauthorized),
		errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})

	// 500 Internal Server Error (默认)
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
