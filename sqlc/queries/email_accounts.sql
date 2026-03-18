-- name: CreateEmailAccount :one
INSERT INTO email_accounts (user_id, email, imap_server, password_enc) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListEmailAccountsByUser :many
SELECT id, user_id, email, imap_server, is_active, last_connected, last_error, created_at FROM email_accounts WHERE user_id = $1 ORDER BY created_at;

-- name: GetActiveEmailAccounts :many
SELECT * FROM email_accounts WHERE is_active = TRUE;

-- name: UpdateEmailAccountStatus :exec
UPDATE email_accounts SET last_connected = $2, last_error = $3 WHERE id = $1;

-- name: DeactivateEmailAccount :exec
UPDATE email_accounts SET is_active = FALSE WHERE id = $1 AND user_id = $2;

-- name: DeleteEmailAccount :exec
DELETE FROM email_accounts WHERE id = $1 AND user_id = $2;

-- name: GetEmailAccountPassword :one
SELECT password_enc FROM email_accounts WHERE id = $1;

-- name: CountEmailAccountsByUser :one
SELECT COUNT(*) FROM email_accounts WHERE user_id = $1;
