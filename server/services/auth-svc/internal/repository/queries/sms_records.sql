-- name: CreateSMSRecord :one
INSERT INTO sms_records (
    id, phone, provider, success, error_msg, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSMSRecordByID :one
SELECT * FROM sms_records
WHERE id = $1 LIMIT 1;

-- name: ListSMSRecordsByPhone :many
SELECT * FROM sms_records
WHERE phone = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSMSRecordsByPhone :one
SELECT COUNT(*) FROM sms_records
WHERE phone = $1;

-- name: CountSMSRecordsByProvider :one
SELECT COUNT(*) FROM sms_records
WHERE provider = $1 AND created_at > $2;

-- name: CountSuccessSMSRecords :one
SELECT COUNT(*) FROM sms_records
WHERE success = true AND created_at > $1;

-- name: CountFailedSMSRecords :one
SELECT COUNT(*) FROM sms_records
WHERE success = false AND created_at > $1;

-- name: DeleteOldSMSRecords :exec
DELETE FROM sms_records
WHERE created_at < $1;
