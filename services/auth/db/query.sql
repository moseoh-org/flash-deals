-- name: CreateUser :one
INSERT INTO auth.users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING id, email, name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM auth.users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM auth.users
WHERE email = $1;

-- name: ExistsUserByEmail :one
SELECT EXISTS(
    SELECT 1 FROM auth.users WHERE email = $1
) AS exists;

-- name: UpdateUser :one
UPDATE auth.users
SET name = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM auth.users
WHERE id = $1;
