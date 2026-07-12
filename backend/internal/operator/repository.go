package operator

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetUserRole(ctx context.Context, userID uuid.UUID) (string, error) {
	var role string
	err := r.pool.QueryRow(ctx, `SELECT role::text FROM users WHERE id = $1`, userID).Scan(&role)
	return role, err
}

func (r *Repository) GetMbararaCityID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM cities WHERE slug = 'mbarara' LIMIT 1`).Scan(&id)
	return id, err
}

func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, operator_type::text, ride_type::text, status::text,
		       wallet_balance, wallet_min_balance
		FROM operator_profiles
		WHERE user_id = $1
	`, userID).Scan(
		&p.ID, &p.UserID, &p.OperatorType, &p.RideType, &p.Status,
		&p.WalletBalance, &p.WalletMin,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, userID uuid.UUID, opType OperatorType, rideType RideType, cityID uuid.UUID) (*Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		INSERT INTO operator_profiles (user_id, operator_type, ride_type, city_id, status)
		VALUES ($1, $2::operator_type, $3::ride_type, $4, 'pending_verification')
		RETURNING id, user_id, operator_type::text, ride_type::text, status::text,
		          wallet_balance, wallet_min_balance
	`, userID, opType, rideType, cityID).Scan(
		&p.ID, &p.UserID, &p.OperatorType, &p.RideType, &p.Status,
		&p.WalletBalance, &p.WalletMin,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
