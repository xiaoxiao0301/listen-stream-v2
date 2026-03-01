-- 创建用户收藏表
CREATE TABLE IF NOT EXISTS favorites (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    song_id VARCHAR(255) NOT NULL,
    song_name VARCHAR(500) NOT NULL,
    singer_name VARCHAR(500) NOT NULL,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, song_id, deleted_at)
);

CREATE INDEX idx_favorites_user_id ON favorites(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_favorites_song_id ON favorites(song_id);
CREATE INDEX idx_favorites_deleted_at ON favorites(deleted_at);

-- 创建播放历史表
CREATE TABLE IF NOT EXISTS play_histories (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    song_id VARCHAR(255) NOT NULL,
    song_name VARCHAR(500) NOT NULL,
    singer_name VARCHAR(500) NOT NULL,
    album_cover VARCHAR(1000),
    duration INTEGER NOT NULL DEFAULT 0,
    played_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_play_histories_user_id ON play_histories(user_id, played_at DESC);
CREATE INDEX idx_play_histories_song_id ON play_histories(song_id);

-- 创建用户歌单表
CREATE TABLE IF NOT EXISTS user_playlists (
    id UUID PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    cover_url VARCHAR(1000),
    song_count INTEGER NOT NULL DEFAULT 0,
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_playlists_user_id ON user_playlists(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_playlists_is_public ON user_playlists(is_public) WHERE deleted_at IS NULL;

-- 创建歌单歌曲关联表
CREATE TABLE IF NOT EXISTS playlist_songs (
    playlist_id UUID NOT NULL REFERENCES user_playlists(id) ON DELETE CASCADE,
    song_id VARCHAR(255) NOT NULL,
    song_name VARCHAR(500) NOT NULL,
    singer_name VARCHAR(500) NOT NULL,
    position INTEGER NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (playlist_id, song_id)
);

CREATE INDEX idx_playlist_songs_playlist_id ON playlist_songs(playlist_id, position);
CREATE INDEX idx_playlist_songs_song_id ON playlist_songs(song_id);

-- 创建触发器：自动更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_playlists_updated_at BEFORE UPDATE ON user_playlists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
