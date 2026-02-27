package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xiaoxiao0301/listen-stream-v2/server/services/auth-svc/internal/domain"
)

// SMSVerificationRepository 短信验证仓储接口
type SMSVerificationRepository interface {
	// Create 创建短信验证
	Create(ctx context.Context, sms *domain.SMSVerification) error
	// GetByID 根据ID获取短信验证
	GetByID(ctx context.Context, id string) (*domain.SMSVerification, error)
	// GetLatest 获取手机号的最新未使用验证码
	GetLatest(ctx context.Context, phone string) (*domain.SMSVerification, error)
	// MarkAsUsed 标记为已使用
	MarkAsUsed(ctx context.Context, id string, usedAt time.Time) error
	// DeleteExpired 删除过期的验证码
	DeleteExpired(ctx context.Context, before time.Time) error
	// CountRecent 统计最近的验证码数量（用于限流）
	CountRecent(ctx context.Context, phone string, after time.Time) (int64, error)
}

// smsVerificationRepository PostgreSQL短信验证仓储实现
type smsVerificationRepository struct {
	db *pgxpool.Pool
}

// NewSMSVerificationRepository 创建短信验证仓储
func NewSMSVerificationRepository(db *pgxpool.Pool) SMSVerificationRepository {
	return &smsVerificationRepository{db: db}
}

// Create 创建短信验证
func (r *smsVerificationRepository) Create(ctx context.Context, sms *domain.SMSVerification) error {
	query := `
		INSERT INTO sms_verifications (id, phone, code, expires_at, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		sms.ID,
		sms.Phone,
		sms.Code,
		sms.ExpiresAt,
		sms.UsedAt,
		sms.CreatedAt,
	)
	return err
}

// GetByID 根据ID获取短信验证
func (r *smsVerificationRepository) GetByID(ctx context.Context, id string) (*domain.SMSVerification, error) {
	query := `
		SELECT id, phone, code, expires_at, used_at, created_at
		FROM sms_verifications
		WHERE id = $1
	`

	var sms domain.SMSVerification
	var usedAt sql.NullTime
	err := r.db.QueryRow(ctx, query, id).Scan(
		&sms.ID,
		&sms.Phone,
		&sms.Code,
		&sms.ExpiresAt,
		&usedAt,
		&sms.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidSMSID
		}
		return nil, err
	}

	if usedAt.Valid {
		sms.UsedAt = &usedAt.Time
	}

	return &sms, nil
}

// GetLatest 获取手机号的最新未使用验证码
func (r *smsVerificationRepository) GetLatest(ctx context.Context, phone string) (*domain.SMSVerification, error) {
	query := `
		SELECT id, phone, code, expires_at, used_at, created_at
		FROM sms_verifications
		WHERE phone = $1 AND used_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	var sms domain.SMSVerification
	var usedAt sql.NullTime
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&sms.ID,
		&sms.Phone,
		&sms.Code,
		&sms.ExpiresAt,
		&usedAt,
		&sms.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidSMSID
		}
		return nil, err
	}

	if usedAt.Valid {
		sms.UsedAt = &usedAt.Time
	}

	return &sms, nil
}

// MarkAsUsed 标记为已使用
func (r *smsVerificationRepository) MarkAsUsed(ctx context.Context, id string, usedAt time.Time) error {
	query := `UPDATE sms_verifications SET used_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, usedAt)
	return err
}

// DeleteExpired 删除过期的验证码
func (r *smsVerificationRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	query := `DELETE FROM sms_verifications WHERE expires_at < $1`
	_, err := r.db.Exec(ctx, query, before)
	return err
}

// CountRecent 统计最近的验证码数量（用于限流）
func (r *smsVerificationRepository) CountRecent(ctx context.Context, phone string, after time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM sms_verifications WHERE phone = $1 AND created_at > $2`
	var count int64
	err := r.db.QueryRow(ctx, query, phone, after).Scan(&count)
	return count, err
}

// ========== SMS Record Repository ==========

// SMSRecordRepository 短信记录仓储接口
type SMSRecordRepository interface {
	// Create 创建短信记录
	Create(ctx context.Context, record *domain.SMSRecord) error
	// GetByID 根据ID获取短信记录
	GetByID(ctx context.Context, id string) (*domain.SMSRecord, error)
	// ListByPhone 分页查询手机号的短信记录
	ListByPhone(ctx context.Context, phone string, limit, offset int) ([]*domain.SMSRecord, error)
	// CountByPhone 统计手机号的短信记录数量
	CountByPhone(ctx context.Context, phone string) (int64, error)
	// CountByProvider 统计提供商的发送数量
	CountByProvider(ctx context.Context, provider string, after time.Time) (int64, error)
	// CountSuccess 统计成功发送数量
	CountSuccess(ctx context.Context, after time.Time) (int64, error)
	// CountFailed 统计失败发送数量
	CountFailed(ctx context.Context, after time.Time) (int64, error)
	// DeleteOld 删除旧记录
	DeleteOld(ctx context.Context, before time.Time) error
}

// smsRecordRepository PostgreSQL短信记录仓储实现
type smsRecordRepository struct {
	db *pgxpool.Pool
}

// NewSMSRecordRepository 创建短信记录仓储
func NewSMSRecordRepository(db *pgxpool.Pool) SMSRecordRepository {
	return &smsRecordRepository{db: db}
}

// Create 创建短信记录
func (r *smsRecordRepository) Create(ctx context.Context, record *domain.SMSRecord) error {
	query := `
		INSERT INTO sms_records (id, phone, provider, success, error_msg, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, query,
		record.ID,
		record.Phone,
		record.Provider,
		record.Success,
		record.ErrorMsg,
		record.CreatedAt,
	)
	return err
}

// GetByID 根据ID获取短信记录
func (r *smsRecordRepository) GetByID(ctx context.Context, id string) (*domain.SMSRecord, error) {
	query := `
		SELECT id, phone, provider, success, error_msg, created_at
		FROM sms_records
		WHERE id = $1
	`

	var record domain.SMSRecord
	err := r.db.QueryRow(ctx, query, id).Scan(
		&record.ID,
		&record.Phone,
		&record.Provider,
		&record.Success,
		&record.ErrorMsg,
		&record.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrInvalidSMSRecordID
		}
		return nil, err
	}
	return &record, nil
}

// ListByPhone 分页查询手机号的短信记录
func (r *smsRecordRepository) ListByPhone(ctx context.Context, phone string, limit, offset int) ([]*domain.SMSRecord, error) {
	query := `
		SELECT id, phone, provider, success, error_msg, created_at
		FROM sms_records
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, phone, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*domain.SMSRecord
	for rows.Next() {
		var record domain.SMSRecord
		err := rows.Scan(
			&record.ID,
			&record.Phone,
			&record.Provider,
			&record.Success,
			&record.ErrorMsg,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// CountByPhone 统计手机号的短信记录数量
func (r *smsRecordRepository) CountByPhone(ctx context.Context, phone string) (int64, error) {
	query := `SELECT COUNT(*) FROM sms_records WHERE phone = $1`
	var count int64
	err := r.db.QueryRow(ctx, query, phone).Scan(&count)
	return count, err
}

// CountByProvider 统计提供商的发送数量
func (r *smsRecordRepository) CountByProvider(ctx context.Context, provider string, after time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM sms_records WHERE provider = $1 AND created_at > $2`
	var count int64
	err := r.db.QueryRow(ctx, query, provider, after).Scan(&count)
	return count, err
}

// CountSuccess 统计成功发送数量
func (r *smsRecordRepository) CountSuccess(ctx context.Context, after time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM sms_records WHERE success = true AND created_at > $1`
	var count int64
	err := r.db.QueryRow(ctx, query, after).Scan(&count)
	return count, err
}

// CountFailed 统计失败发送数量
func (r *smsRecordRepository) CountFailed(ctx context.Context, after time.Time) (int64, error) {
	query := `SELECT COUNT(*) FROM sms_records WHERE success = false AND created_at > $1`
	var count int64
	err := r.db.QueryRow(ctx, query, after).Scan(&count)
	return count, err
}

// DeleteOld 删除旧记录
func (r *smsRecordRepository) DeleteOld(ctx context.Context, before time.Time) error {
	query := `DELETE FROM sms_records WHERE created_at < $1`
	_, err := r.db.Exec(ctx, query, before)
	return err
}
