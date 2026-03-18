-- name: CreateEmailCode :one
INSERT INTO email_codes (email_account_id, user_id, sender, subject, code, rule_name, raw_body_hash, tg_message_id, received_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: ListRecentCodesByUser :many
SELECT ec.*, ea.email FROM email_codes ec JOIN email_accounts ea ON ec.email_account_id = ea.id WHERE ec.user_id = $1 ORDER BY ec.received_at DESC LIMIT $2;

-- name: CodeExistsByBodyHash :one
SELECT EXISTS(SELECT 1 FROM email_codes WHERE raw_body_hash = $1) AS exists;

-- name: DeleteOldCodes :execrows
DELETE FROM email_codes WHERE created_at < NOW() - INTERVAL '24 hours';
