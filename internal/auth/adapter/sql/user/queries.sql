-- name: CreateUser :exec
INSERT INTO auth.user (id, email, password_hash) VALUES ($1, $2, $3);

-- name: GetUserByEmail :one
SELECT * FROM auth.user WHERE email = $1;