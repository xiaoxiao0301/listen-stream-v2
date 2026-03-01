package domain

import "time"

// PlaylistSong 歌单-歌曲关联实体
type PlaylistSong struct {
	PlaylistID string    `json:"playlist_id"` // 歌单ID
	SongID     string    `json:"song_id"`     // 歌曲ID
	SongName   string    `json:"song_name"`   // 歌名（冗余存储）
	SingerName string    `json:"singer_name"` // 歌手名（冗余存储）
	Position   int       `json:"position"`    // 排序位置
	AddedAt    time.Time `json:"added_at"`    // 添加时间
}

// Validate 验证歌单歌曲关联数据
func (ps *PlaylistSong) Validate() error {
	if ps.PlaylistID == "" {
		return ErrInvalidPlaylistID
	}
	if ps.SongID == "" {
		return ErrInvalidSongID
	}
	if ps.Position < 0 {
		return ErrInvalidPosition
	}
	return nil
}
