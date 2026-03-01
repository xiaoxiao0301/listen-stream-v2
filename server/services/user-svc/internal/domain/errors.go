package domain

import "errors"

var (
	// 通用错误
	ErrInvalidUserID   = errors.New("invalid user id")
	ErrInvalidSongID   = errors.New("invalid song id")
	ErrInvalidSongName = errors.New("invalid song name")
	
	// 收藏相关错误
	ErrFavoriteNotFound      = errors.New("favorite not found")
	ErrFavoriteAlreadyExists = errors.New("favorite already exists")
	
	// 播放历史相关错误
	ErrHistoryNotFound   = errors.New("history not found")
	ErrInvalidDuration   = errors.New("invalid duration")
	
	// 歌单相关错误
	ErrPlaylistNotFound           = errors.New("playlist not found")
	ErrPlaylistAlreadyExists      = errors.New("playlist already exists")
	ErrInvalidPlaylistID          = errors.New("invalid playlist id")
	ErrInvalidPlaylistName        = errors.New("invalid playlist name")
	ErrPlaylistNameTooLong        = errors.New("playlist name too long")
	ErrPlaylistDescriptionTooLong = errors.New("playlist description too long")
	ErrInvalidPosition            = errors.New("invalid position")
	ErrSongAlreadyInPlaylist      = errors.New("song already in playlist")
	ErrSongNotInPlaylist          = errors.New("song not in playlist")
	
	// 权限相关错误
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
