package upstream

import (
	"context"
	"time"
)

// ===== 通用响应结构 =====

// APIResponse 通用API响应
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ===== Home模块 =====

// Banner 轮播图
type Banner struct {
	ID       string `json:"id"`
	PicURL   string `json:"pic_url"`
	LinkURL  string `json:"link_url"`
	Title    string `json:"title"`
	Type     int    `json:"type"`
}

// Playlist 歌单
type Playlist struct {
	DissID      string `json:"dissid"`
	DissName    string `json:"diss_name"`
	Logo        string `json:"logo"`
	CreateTime  string `json:"create_time"`
	ListenNum   int64  `json:"listen_num"`
	SongCount   int    `json:"song_count"`
	Description string `json:"description"`
	Creator     string `json:"creator"`
}

// Song 歌曲 (标准化结构，兼容多个上游源)
type Song struct {
	// QQ Music style
	SongID     string   `json:"song_id,omitempty"`
	SongMid    string   `json:"song_mid,omitempty"`
	SongName   string   `json:"song_name,omitempty"`
	AlbumID    string   `json:"album_id,omitempty"`
	AlbumMid   string   `json:"album_mid,omitempty"`
	AlbumName  string   `json:"album_name,omitempty"`
	SingerID   string   `json:"singer_id,omitempty"`
	SingerMid  string   `json:"singer_mid,omitempty"`
	SingerName string   `json:"singer_name,omitempty"`
	Singers    []Singer `json:"singers,omitempty"`
	Duration   int      `json:"duration,omitempty"`
	PicURL     string   `json:"pic_url,omitempty"`
	
	// Generic standardized fields (for fallback sources)
	ID       string `json:"id,omitempty"`       // Universal song ID
	Name     string `json:"name,omitempty"`     // Universal song name
	Artist   string `json:"artist,omitempty"`   // Universal artist name
	Album    string `json:"album,omitempty"`    // Universal album name
	CoverURL string `json:"cover_url,omitempty"` // Universal cover URL
}

// Album 专辑
type Album struct {
	AlbumID     string `json:"album_id"`
	AlbumMid    string `json:"album_mid"`
	AlbumName   string `json:"album_name"`
	AlbumPic    string `json:"album_pic"`
	PublicTime  string `json:"public_time"`
	SingerID    string `json:"singer_id"`
	SingerMid   string `json:"singer_mid"`
	SingerName  string `json:"singer_name"`
	SongCount   int    `json:"song_count"`
	Description string `json:"description"`
}

// ===== Singer模块 =====

// Singer 歌手
type Singer struct {
	SingerID   string `json:"singer_id"`
	SingerMid  string `json:"singer_mid"`
	SingerName string `json:"singer_name"`
	SingerPic  string `json:"singer_pic"`
	Country    string `json:"country"`
	SongCount  int    `json:"song_count"`
	AlbumCount int    `json:"album_count"`
	MVCount    int    `json:"mv_count"`
}

// CategoryFilter 分类筛选器
type CategoryFilter struct {
	Key   string              `json:"key"`
	Name  string              `json:"name"`
	Items []CategoryFilterItem `json:"items"`
}

// CategoryFilterItem 筛选项
type CategoryFilterItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ===== Ranking模块 =====

// RankingCategory 排行榜分类
type RankingCategory struct {
	GroupID   int            `json:"group_id"`
	GroupName string         `json:"group_name"`
	List      []RankingItem  `json:"list"`
}

// RankingItem 排行榜项
type RankingItem struct {
	TopID      int    `json:"topId"`
	Title      string `json:"title"`
	PicURL     string `json:"pic_url"`
	Period     string `json:"period"`
	UpdateTime string `json:"update_time"`
	ListenNum  int64  `json:"listen_num"`
	Songs      []Song `json:"songs"` // 前三首歌曲
}

// ===== MV模块 =====

// MV MV信息
type MV struct {
	VID        string `json:"vid"`
	Title      string `json:"title"`
	PicURL     string `json:"pic_url"`
	SingerID   string `json:"singer_id"`
	SingerMid  string `json:"singer_mid"`
	SingerName string `json:"singer_name"`
	Duration   int    `json:"duration"`
	PublishTime string `json:"publish_time"`
	PlayCount  int64  `json:"play_count"`
}

// MVDetail MV详情
type MVDetail struct {
	MV
	URL         string `json:"url"`
	Description string `json:"description"`
}

// MVCategory MV分类
type MVCategory struct {
	Area    []CategoryFilterItem `json:"area"`
	Version []CategoryFilterItem `json:"version"`
}

// ===== Radio模块 =====

// Radio 电台
type Radio struct {
	ID          int    `json:"id"`
	RadioName   string `json:"radio_name"`
	RadioPic    string `json:"radio_pic"`
	Description string `json:"description"`
}

// ===== Song模块 =====

// SongDetail 歌曲详情
type SongDetail struct {
	Song
	Lyric       string `json:"lyric"`
	TransLyric  string `json:"trans_lyric"`
	Album       Album  `json:"album"`
	PublishTime string `json:"publish_time"`
}

// SongURL 歌曲播放URL
type SongURL struct {
	URL      string `json:"url"`
	SongMid  string `json:"songmid"`
	VKey     string `json:"vkey"`
	Filename string `json:"filename"`
	Source   string `json:"source"`
}

// ===== Lyric模块 =====

// Lyric 歌词
type Lyric struct {
	Lyric      string `json:"lyric"`
	TransLyric string `json:"trans_lyric"`
}

// ===== Search模块 =====

// SearchResult 搜索结果
type SearchResult struct {
	Songs    []Song     `json:"songs,omitempty"`
	Singers  []Singer   `json:"singers,omitempty"`
	Albums   []Album    `json:"albums,omitempty"`
	MVs      []MV       `json:"mvs,omitempty"`
	Total    int        `json:"total"`
}

// HotKey 热搜关键词
type HotKey struct {
	Keyword string `json:"keyword"`
	Score   int    `json:"score"`
}

// ===== Playlist模块 =====

// PlaylistCategory 歌单分类
type PlaylistCategory struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
	ParentID int    `json:"parent_id"`
}

// PlaylistDetail 歌单详情
type PlaylistDetail struct {
	Playlist
	Songs []Song `json:"songs"`
}

// ===== Page分页 =====

// PageRequest 分页请求
type PageRequest struct {
	Page int `json:"page" form:"page"`
	Size int `json:"size" form:"size"`
}

// PageResponse 分页响应
type PageResponse struct {
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	Data  interface{} `json:"data"`
}

// ===== Fallback相关 =====

// FallbackSongInfo Fallback歌曲信息
type FallbackSongInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Artist   []string `json:"artist"`
	Album    string   `json:"album"`
	PicID    string   `json:"pic_id"`
	URLID    string   `json:"url_id"`
	LyricID  string   `json:"lyric_id"`
	Source   string   `json:"source"`
	From     string   `json:"from"`
}

// FallbackSongURL Fallback歌曲URL
type FallbackSongURL struct {
	URL  string `json:"url"`
	BR   int    `json:"br"`
	Size int64  `json:"size"`
	From string `json:"from"`
}

// ===== Config配置 =====

// UpstreamConfig 上游配置
type UpstreamConfig struct {
	Name        string
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	RateLimit   int
	Cookie      string
	FallbackURL string // Fallback API URL
}

// SongURLResponse 标准化的歌曲URL响应
type SongURLResponse struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	Size    int64  `json:"size"`
	Format  string `json:"format"`
}

// ClientInterface 上游客户端接口
type ClientInterface interface {
	// 播放和搜索（用于 FallbackManager）
	GetSongURL(ctx context.Context, songMid, songName string) (*SongURL, error)
	SearchSong(ctx context.Context, songName, artistName string) (*SearchResult, error)
	GetBanners(ctx context.Context) ([]Banner, error)
	HealthCheck(ctx context.Context) error

	// Home模块
	GetDailyRecommendPlaylists(ctx context.Context) ([]Playlist, error)
	GetRecommendPlaylists(ctx context.Context) ([]Playlist, error)
	GetNewSongs(ctx context.Context, typ int) ([]Song, error)
	GetNewAlbums(ctx context.Context, typ int) ([]Album, error)

	// Playlist模块
	GetPlaylistCategories(ctx context.Context) ([]PlaylistCategory, error)
	GetPlaylistsByCategory(ctx context.Context, req GetPlaylistsByCategoryRequest) (*PageResponse, error)
	GetPlaylistDetail(ctx context.Context, dissID string) (*PlaylistDetail, error)

	// Song模块
	GetSongDetail(ctx context.Context, songMid string) (*SongDetail, error)
	GetLyric(ctx context.Context, songMid string) (*Lyric, error)

	// Album模块
	GetAlbumDetail(ctx context.Context, albumMid string) (*Album, error)
	GetAlbumSongs(ctx context.Context, albumMid string) ([]Song, error)

	// Search模块
	GetHotKeys(ctx context.Context) ([]HotKey, error)
	SearchSongs(ctx context.Context, keyword string, page, size int) (*PageResponse, error)
	SearchSingers(ctx context.Context, keyword string, page, size int) (*PageResponse, error)
	SearchAlbums(ctx context.Context, keyword string, page, size int) (*PageResponse, error)
	SearchMVs(ctx context.Context, keyword string, page, size int) (*PageResponse, error)

	// Singer模块
	GetSingerCategories(ctx context.Context) ([]CategoryFilter, error)
	GetSingerList(ctx context.Context, req GetSingerListRequest) (*PageResponse, error)
	GetSingerDetail(ctx context.Context, singerMid string, page int) (*PageResponse, error)
	GetSingerAlbums(ctx context.Context, singerMid string, page, size int) (*PageResponse, error)
	GetSingerMVs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error)
	GetSingerSongs(ctx context.Context, singerMid string, page, size int) (*PageResponse, error)

	// Ranking模块
	GetRankingList(ctx context.Context) ([]RankingCategory, error)
	GetRankingDetail(ctx context.Context, topID, page, size int, period string) (*PageResponse, error)

	// Radio模块
	GetRadioCategories(ctx context.Context) ([]Radio, error)
	GetRadioSongs(ctx context.Context, radioID int) ([]Song, error)

	// MV模块
	GetMVCategories(ctx context.Context) (*MVCategory, error)
	GetMVList(ctx context.Context, area, version, page, size int) (*PageResponse, error)
	GetMVDetail(ctx context.Context, vid string) (*MVDetail, error)
}
