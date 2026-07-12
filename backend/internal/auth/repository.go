package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID    uuid.UUID
	Phone string
	Role  string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SaveOTP(ctx context.Context, phone, code string, expiresAt time.Time) error {
	hash := hashOTP(code)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO otp_codes (phone, code_hash, expires_at)
		VALUES ($1, $2, $3)
	`, phone, hash, expiresAt)
	return err
}

func (r *Repository) VerifyOTP(ctx context.Context, phone, code string) (bool, error) {
	hash := hashOTP(code)
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM otp_codes
		WHERE phone = $1
		  AND code_hash = $2
		  AND used_at IS NULL
		  AND expires_at > now()
		ORDER BY created_at DESC
		LIMIT 1
	`, phone, hash).Scan(&id)
	if err != nil {
		return false, err
	}

	_, err = r.pool.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) GetUserByPhone(ctx context.Context, phone string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, phone, role::text FROM users WHERE phone = $1
	`, phone).Scan(&u.ID, &u.Phone, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(ctx context.Context, phone, role string) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var u User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (phone, role)
		VALUES ($1, $2::user_role)
		RETURNING id, phone, role::text
	`, phone, role).Scan(&u.ID, &u.Phone, &u.Role)
	if err != nil {
		return nil, err
	}

	if role == "passenger" {
		_, err = tx.Exec(ctx, `
			INSERT INTO passenger_profiles (user_id) VALUES ($1)
			ON CONFLICT (user_id) DO NOTHING
		`, u.ID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}

func GenerateOTP(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	format := fmt.Sprintf("%%0%dd", length)
	return fmt.Sprintf(format, n.Int64()), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, phone, role::text FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Phone, &u.Role)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func hashOTP(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
