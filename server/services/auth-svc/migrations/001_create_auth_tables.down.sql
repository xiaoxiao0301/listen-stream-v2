-- 001_create_auth_tables.down.sql
-- 回滚认证服务数据库表

DROP TABLE IF EXISTS sms_records;
DROP TABLE IF EXISTS sms_verifications;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;
