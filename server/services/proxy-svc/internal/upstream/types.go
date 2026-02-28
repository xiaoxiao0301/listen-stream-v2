package upstream

import "time"

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

// Song 歌曲
type Song struct {
	SongID     string   `json:"song_id"`
	SongMid    string   `json:"song_mid"`
	SongName   string   `json:"song_name"`
	AlbumID    string   `json:"album_id"`
	AlbumMid   string   `json:"album_mid"`
	AlbumName  string   `json:"album_name"`
	SingerID   string   `json:"singer_id"`
	SingerMid  string   `json:"singer_mid"`
	SingerName string   `json:"singer_name"`
	Singers    []Singer `json:"singers"`
	Duration   int      `json:"duration"`
	PicURL     string   `json:"pic_url"`
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
	Name            string
	BaseURL         string
	Timeout         time.Duration
	MaxRetries      int
	RateLimit       int
	Cookie          string
	FallbackURL     string // Fallback API URL
	BreakerSettings BreakerSettings
}
