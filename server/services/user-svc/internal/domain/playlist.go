package domain

import "time"

// UserPlaylist 用户歌单实体
type UserPlaylist struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CoverURL    string     `json:"cover_url"`
	SongCount   int        `json:"song_count"`
	IsPublic    bool       `json:"is_public"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Validate 验证歌单数据
func (p *UserPlaylist) Validate() error {
	if p.UserID == "" {
		return ErrInvalidUserID
	}
	if p.Name == "" {
		return ErrInvalidPlaylistName
	}
	if len(p.Name) > 100 {
		return ErrPlaylistNameTooLong
	}
	if len(p.Description) > 500 {
		return ErrPlaylistDescriptionTooLong
	}
	return nil
}

// IsDeleted 判断是否已删除
func (p *UserPlaylist) IsDeleted() bool {
	return p.DeletedAt != nil
}

// SoftDelete 软删除
func (p *UserPlaylist) SoftDelete() {
	now := time.Now()
	p.DeletedAt = &now
}

// Restore 恢复
func (p *UserPlaylist) Restore() {
	p.DeletedAt = nil
}

// ValidatePlaylistName 验证歌单名称
func ValidatePlaylistName(name string) error {
	if name == "" {
		return ErrInvalidPlaylistName
	}
	if len(name) > 100 {
		return ErrPlaylistNameTooLong
	}
	return nil
}

// SetName 设置歌单名称
func (p *UserPlaylist) SetName(name string) {
	p.Name = name
}

// IncrementSongCount 增加歌曲数量
func (p *UserPlaylist) IncrementSongCount() {
	p.SongCount++
	p.UpdatedAt = time.Now()
}

// DecrementSongCount 减少歌曲数量
func (p *UserPlaylist) DecrementSongCount() {
	if p.SongCount > 0 {
		p.SongCount--
	}
	p.UpdatedAt = time.Now()
}
