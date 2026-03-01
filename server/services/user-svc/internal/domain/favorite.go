package domain

import "time"

// Favorite 收藏实体
type Favorite struct {
	ID        string    `json:"id"`         // UUID
	UserID    string    `json:"user_id"`    // 用户ID
	SongID    string    `json:"song_id"`    // 歌曲ID
	SongName  string    `json:"song_name"`  // 歌名（冗余存储，支持离线显示）
	SingerName string   `json:"singer_name"` // 歌手名（冗余存储）
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // 软删除时间
	CreatedAt time.Time `json:"created_at"` // 创建时间
}

// IsDeleted 判断是否已删除
func (f *Favorite) IsDeleted() bool {
	return f.DeletedAt != nil
}

// SoftDelete 软删除
func (f *Favorite) SoftDelete() {
	now := time.Now()
	f.DeletedAt = &now
}

// Restore 恢复
func (f *Favorite) Restore() {
	f.DeletedAt = nil
}
