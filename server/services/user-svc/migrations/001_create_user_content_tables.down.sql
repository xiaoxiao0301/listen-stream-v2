-- 删除触发器
DROP TRIGGER IF EXISTS update_user_playlists_updated_at ON user_playlists;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 删除表
DROP TABLE IF EXISTS playlist_songs;
DROP TABLE IF EXISTS user_playlists;
DROP TABLE IF EXISTS play_histories;
DROP TABLE IF EXISTS favorites;
