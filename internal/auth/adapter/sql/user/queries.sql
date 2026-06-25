-- name: IsEmailTaken :one
SELECT EXISTS (
    SELECT 1 FROM auth.user
    WHERE email = $1
);

-- name: CreateUser :exec
INSERT INTO auth.user (id, email, password_hash) VALUES ($1, $2, $3);

-- name: GetUserByEmail :one
SELECT * FROM auth.user WHERE email = $1;