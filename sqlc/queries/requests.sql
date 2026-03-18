-- name: CreateRequest :one
INSERT INTO requests (user_id, expense_type, payment_method, address, address_photo, amount, antique_account, comment, flow_type, tg_message_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING *;

-- name: GetRequestByID :one
SELECT * FROM requests WHERE id = $1;

-- name: ListRequestsByUser :many
SELECT * FROM requests WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListPendingRequests :many
SELECT r.*, u.username, u.first_name FROM requests r JOIN users u ON r.user_id = u.id WHERE r.status = 'pending' ORDER BY r.created_at ASC;

-- name: UpdateRequestStatus :one
UPDATE requests SET status = $2 WHERE id = $1 RETURNING *;

-- name: CountRequestsByStatus :many
SELECT status, COUNT(*) as count FROM requests GROUP BY status;
