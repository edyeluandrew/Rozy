package verification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotOperator       = errors.New("operator profile required")
	ErrPendingExists     = errors.New("verification already pending review")
	ErrAlreadyApproved   = errors.New("operator already verified")
	ErrNINTaken          = errors.New("nin already registered")
	ErrInvalidSubmission = errors.New("invalid verification submission")
)

type DocumentInput struct {
	DocType    string `json:"doc_type"`
	StorageKey string `json:"storage_key"`
	SHA256Hash string `json:"sha256_hash"`
	MimeType   string `json:"mime_type"`
}

type SubmitInput struct {
	LegalName       string          `json:"legal_name"`
	NIN             string          `json:"nin"`
	PermitNumber    string          `json:"permit_number"`
	PermitExpiry    string          `json:"permit_expiry"`
	InsuranceExpiry string          `json:"insurance_expiry"`
	Plate           string          `json:"plate"`
	BikeMake        string          `json:"bike_make,omitempty"`
	BikeModel       string          `json:"bike_model,omitempty"`
	BikeColor       string          `json:"bike_color,omitempty"`
	CarMake         string          `json:"car_make,omitempty"`
	CarModel        string          `json:"car_model,omitempty"`
	CarColor        string          `json:"car_color,omitempty"`
	CarCapacity     int             `json:"car_capacity,omitempty"`
	Documents       []DocumentInput `json:"documents"`
}

type Status struct {
	HasSubmission bool   `json:"has_submission"`
	Status        string `json:"status,omitempty"`
	SubmissionID  string `json:"submission_id,omitempty"`
	SubmittedAt   string `json:"submitted_at,omitempty"`
	RejectionNote string `json:"rejection_reason,omitempty"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func HashNIN(nin string) string {
	sum := sha256.Sum256([]byte(strings.ToUpper(strings.TrimSpace(nin))))
	return hex.EncodeToString(sum[:])
}

func last4NIN(nin string) string {
	n := strings.ToUpper(strings.TrimSpace(nin))
	if len(n) >= 4 {
		return n[len(n)-4:]
	}
	return n
}

func (r *Repository) GetOperatorByUser(ctx context.Context, userID uuid.UUID) (operatorID uuid.UUID, rideType string, status string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT id, ride_type::text, status::text
		FROM operator_profiles WHERE user_id = $1
	`, userID).Scan(&operatorID, &rideType, &status)
	return
}

func (r *Repository) LatestSubmission(ctx context.Context, operatorID uuid.UUID) (*Status, error) {
	var s Status
	var id uuid.UUID
	var submitted time.Time
	var rejection *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, status::text, submitted_at, rejection_reason
		FROM verification_submissions
		WHERE operator_profile_id = $1
		ORDER BY submitted_at DESC LIMIT 1
	`, operatorID).Scan(&id, &s.Status, &submitted, &rejection)
	if errors.Is(err, pgx.ErrNoRows) {
		return &Status{HasSubmission: false}, nil
	}
	if err != nil {
		return nil, err
	}
	s.HasSubmission = true
	s.SubmissionID = id.String()
	s.SubmittedAt = submitted.UTC().Format(time.RFC3339)
	if rejection != nil {
		s.RejectionNote = *rejection
	}
	return &s, nil
}

func (r *Repository) Submit(ctx context.Context, operatorID uuid.UUID, rideType string, in SubmitInput, ninHash string) (uuid.UUID, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	var submissionID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO verification_submissions (operator_profile_id, status)
		VALUES ($1, 'pending')
		RETURNING id
	`, operatorID).Scan(&submissionID)
	if err != nil {
		return uuid.Nil, err
	}

	for _, doc := range in.Documents {
		_, err = tx.Exec(ctx, `
			INSERT INTO documents (submission_id, doc_type, storage_key, sha256_hash, mime_type)
			VALUES ($1, $2::doc_type, $3, $4, $5)
		`, submissionID, doc.DocType, doc.StorageKey, doc.SHA256Hash, doc.MimeType)
		if err != nil {
			return uuid.Nil, err
		}
	}

	permitExpiry, _ := time.Parse("2006-01-02", in.PermitExpiry)
	insuranceExpiry, _ := time.Parse("2006-01-02", in.InsuranceExpiry)

	_, err = tx.Exec(ctx, `
		UPDATE operator_profiles
		SET full_name = $2, nin_hash = $3, nin_last4 = $4, updated_at = now()
		WHERE id = $1
	`, operatorID, in.LegalName, ninHash, last4NIN(in.NIN))
	if err != nil {
		return uuid.Nil, err
	}

	switch rideType {
	case "boda":
		_, err = tx.Exec(ctx, `
			INSERT INTO boda_details (operator_profile_id, plate, bike_make, bike_model, bike_color, permit_number, permit_expiry)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (operator_profile_id) DO UPDATE SET
			  plate = EXCLUDED.plate, bike_make = EXCLUDED.bike_make, bike_model = EXCLUDED.bike_model,
			  bike_color = EXCLUDED.bike_color, permit_number = EXCLUDED.permit_number, permit_expiry = EXCLUDED.permit_expiry
		`, operatorID, strings.ToUpper(in.Plate), in.BikeMake, in.BikeModel, in.BikeColor, in.PermitNumber, permitExpiry)
	case "car_basic", "car_xl":
		category := "basic"
		capacity := in.CarCapacity
		if rideType == "car_xl" {
			category = "xl"
			if capacity < 5 {
				capacity = 5
			}
		} else if capacity == 0 {
			capacity = 4
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO car_details (operator_profile_id, plate, category, capacity, make, model, color, permit_number, permit_expiry)
			VALUES ($1, $2, $3::car_category, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (operator_profile_id) DO UPDATE SET
			  plate = EXCLUDED.plate, category = EXCLUDED.category, capacity = EXCLUDED.capacity,
			  make = EXCLUDED.make, model = EXCLUDED.model, color = EXCLUDED.color,
			  permit_number = EXCLUDED.permit_number, permit_expiry = EXCLUDED.permit_expiry
		`, operatorID, strings.ToUpper(in.Plate), category, capacity, in.CarMake, in.CarModel, in.CarColor, in.PermitNumber, permitExpiry)
	default:
		return uuid.Nil, fmt.Errorf("unknown ride type")
	}
	if err != nil {
		return uuid.Nil, err
	}

	_ = insuranceExpiry // stored on approval snapshot later
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return submissionID, nil
}

func (r *Repository) NINExists(ctx context.Context, ninHash string, excludeOperator uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM operator_profiles
			WHERE nin_hash = $1 AND id <> $2
		)
	`, ninHash, excludeOperator).Scan(&exists)
	return exists, err
}

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key")
}
