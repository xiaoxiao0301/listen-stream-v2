package domain

import "time"

// PlayHistory 播放历史实体
// 约束: 每用户最多500条，超出自动删除最早记录
type PlayHistory struct {
	ID         string    `json:"id"`          // UUID
	UserID     string    `json:"user_id"`     // 用户ID
	SongID     string    `json:"song_id"`     // 歌曲ID
	SongName   string    `json:"song_name"`   // 歌名（冗余存储）
	SingerName string    `json:"singer_name"` // 歌手名（冗余存储）
	AlbumCover string    `json:"album_cover"` // 专辑封面URL（冗余存储）
	Duration   int       `json:"duration"`    // 播放时长（秒）
	PlayedAt   time.Time `json:"played_at"`   // 播放时间
	CreatedAt  time.Time `json:"created_at"`  // 创建时间
}

// Validate 验证播放历史数据
func (h *PlayHistory) Validate() error {
	if h.UserID == "" {
		return ErrInvalidUserID
	}
	if h.SongID == "" {
		return ErrInvalidSongID
	}
	if h.SongName == "" {
		return ErrInvalidSongName
	}
	if h.Duration < 0 {
		return ErrInvalidDuration
	}
	return nil
}
