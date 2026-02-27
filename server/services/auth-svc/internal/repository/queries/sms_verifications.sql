-- name: CreateSMSVerification :one
INSERT INTO sms_verifications (
    id, phone, code, expires_at, used_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSMSVerificationByID :one
SELECT * FROM sms_verifications
WHERE id = $1 LIMIT 1;

-- name: GetLatestSMSVerification :one
SELECT * FROM sms_verifications
WHERE phone = $1 AND used_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkSMSVerificationUsed :exec
UPDATE sms_verifications
SET used_at = $2
WHERE id = $1;

-- name: DeleteExpiredSMSVerifications :exec
DELETE FROM sms_verifications
WHERE expires_at < $1;

-- name: CountSMSVerificationsByPhone :one
SELECT COUNT(*) FROM sms_verifications
WHERE phone = $1 AND created_at > $2;
