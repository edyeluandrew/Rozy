package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrNotPending         = errors.New("submission is not pending")
)

type QueueItem struct {
	SubmissionID string `json:"submission_id"`
	OperatorID   string `json:"operator_id"`
	LegalName    string `json:"legal_name"`
	Phone        string `json:"phone"`
	RideType     string `json:"ride_type"`
	Plate        string `json:"plate"`
	Status       string `json:"status"`
	SubmittedAt  string `json:"submitted_at"`
}

type Document struct {
	ID         string `json:"id"`
	DocType    string `json:"doc_type"`
	StorageKey string `json:"storage_key"`
	MimeType   string `json:"mime_type"`
}

type SubmissionDetail struct {
	QueueItem
	PermitNumber    string     `json:"permit_number"`
	PermitExpiry    string     `json:"permit_expiry,omitempty"`
	InsuranceExpiry string     `json:"insurance_expiry,omitempty"`
	NINLast4        string     `json:"nin_last4,omitempty"`
	Documents       []Document `json:"documents"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListQueue(ctx context.Context, status string) ([]QueueItem, error) {
	if status == "" {
		status = "pending"
	}
	rows, err := r.pool.Query(ctx, `
		SELECT vs.id, op.id, COALESCE(op.full_name,''), u.phone, op.ride_type::text,
		       COALESCE(bd.plate, cd.plate, ''), vs.status::text, vs.submitted_at
		FROM verification_submissions vs
		JOIN operator_profiles op ON op.id = vs.operator_profile_id
		JOIN users u ON u.id = op.user_id
		LEFT JOIN boda_details bd ON bd.operator_profile_id = op.id
		LEFT JOIN car_details cd ON cd.operator_profile_id = op.id
		WHERE vs.status = $1::verification_status
		ORDER BY vs.submitted_at ASC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		var subID, opID uuid.UUID
		var submitted time.Time
		if err := rows.Scan(&subID, &opID, &item.LegalName, &item.Phone, &item.RideType,
			&item.Plate, &item.Status, &submitted); err != nil {
			return nil, err
		}
		item.SubmissionID = subID.String()
		item.OperatorID = opID.String()
		item.SubmittedAt = submitted.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetSubmission(ctx context.Context, submissionID uuid.UUID) (*SubmissionDetail, error) {
	var d SubmissionDetail
	var subID, opID uuid.UUID
	var submitted time.Time
	var permitExpiry *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT vs.id, op.id, COALESCE(op.full_name,''), u.phone, op.ride_type::text,
		       COALESCE(bd.plate, cd.plate, ''), vs.status::text, vs.submitted_at,
		       COALESCE(bd.permit_number, cd.permit_number, ''),
		       COALESCE(bd.permit_expiry, cd.permit_expiry),
		       op.nin_last4
		FROM verification_submissions vs
		JOIN operator_profiles op ON op.id = vs.operator_profile_id
		JOIN users u ON u.id = op.user_id
		LEFT JOIN boda_details bd ON bd.operator_profile_id = op.id
		LEFT JOIN car_details cd ON cd.operator_profile_id = op.id
		WHERE vs.id = $1
	`, submissionID).Scan(
		&subID, &opID, &d.LegalName, &d.Phone, &d.RideType, &d.Plate, &d.Status, &submitted,
		&d.PermitNumber, &permitExpiry, &d.NINLast4,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}

	d.SubmissionID = subID.String()
	d.OperatorID = opID.String()
	d.SubmittedAt = submitted.UTC().Format(time.RFC3339)
	if permitExpiry != nil {
		d.PermitExpiry = permitExpiry.Format("2006-01-02")
	}

	docRows, err := r.pool.Query(ctx, `
		SELECT id, doc_type::text, storage_key, COALESCE(mime_type,'')
		FROM documents WHERE submission_id = $1 ORDER BY uploaded_at
	`, submissionID)
	if err != nil {
		return nil, err
	}
	defer docRows.Close()

	for docRows.Next() {
		var doc Document
		var id uuid.UUID
		if err := docRows.Scan(&id, &doc.DocType, &doc.StorageKey, &doc.MimeType); err != nil {
			return nil, err
		}
		doc.ID = id.String()
		d.Documents = append(d.Documents, doc)
	}

	return &d, docRows.Err()
}

func (r *Repository) Approve(ctx context.Context, submissionID, adminID uuid.UUID) error {
	detail, err := r.GetSubmission(ctx, submissionID)
	if err != nil {
		return err
	}
	if detail.Status != "pending" {
		return ErrNotPending
	}

	opID, _ := uuid.Parse(detail.OperatorID)
	insuranceExpiry := time.Now().AddDate(1, 0, 0) // fallback; admin verifies manually in MVP
	permitExpiry, _ := time.Parse("2006-01-02", detail.PermitExpiry)

	faceKey := ""
	var vehicleKeys []string
	for _, doc := range detail.Documents {
		if doc.DocType == "selfie" && faceKey == "" {
			faceKey = doc.StorageKey
		}
		if strings.HasPrefix(doc.DocType, "vehicle") || doc.DocType == "plate_closeup" {
			vehicleKeys = append(vehicleKeys, doc.StorageKey)
		}
	}
	if faceKey == "" && len(detail.Documents) > 0 {
		faceKey = detail.Documents[0].StorageKey
	}
	if vehicleKeys == nil {
		vehicleKeys = []string{}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE verification_submissions
		SET status = 'approved', reviewed_by = $2, reviewed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, submissionID, adminID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO verified_operator_snapshots (
			operator_profile_id, submission_id, legal_name, nin_last4, plate, ride_type,
			face_photo_key, vehicle_photo_keys, permit_expiry, insurance_expiry, approved_by
		)
		SELECT $1, $2, $3, COALESCE($4,''), $5, op.ride_type, $6, $7, $8, $9, $10
		FROM operator_profiles op WHERE op.id = $1
	`, opID, submissionID, detail.LegalName, detail.NINLast4, detail.Plate,
		faceKey, vehicleKeys, permitExpiry, insuranceExpiry, adminID)
	if err != nil {
		return fmt.Errorf("snapshot: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE operator_profiles SET status = 'offline', updated_at = now() WHERE id = $1
	`, opID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) Reject(ctx context.Context, submissionID, adminID uuid.UUID, reason string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE verification_submissions
		SET status = 'rejected', rejection_reason = $3, reviewed_by = $2, reviewed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, submissionID, adminID, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotPending
	}
	return nil
}

func (r *Repository) PendingCount(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM verification_submissions WHERE status = 'pending'
	`).Scan(&n)
	return n, err
}

type ActiveTripItem struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	RideType        string   `json:"ride_type"`
	EstimatedFare   *int64   `json:"estimated_fare,omitempty"`
	PassengerPhone  string   `json:"passenger_phone"`
	DriverName      string   `json:"driver_name,omitempty"`
	DriverPlate     string   `json:"driver_plate,omitempty"`
	PickupLat       float64  `json:"pickup_lat"`
	PickupLng       float64  `json:"pickup_lng"`
	DriverLat       *float64 `json:"driver_lat,omitempty"`
	DriverLng       *float64 `json:"driver_lng,omitempty"`
	CreatedAt       string   `json:"created_at"`
	AssignedAt      string   `json:"assigned_at,omitempty"`
}

type OperatorListItem struct {
	ID            string `json:"id"`
	Phone         string `json:"phone"`
	Name          string `json:"name"`
	RideType      string `json:"ride_type"`
	Status        string `json:"status"`
	Plate         string `json:"plate,omitempty"`
	WalletBalance int64  `json:"wallet_balance"`
	WalletMin     int64  `json:"wallet_min_balance"`
	Verified      bool   `json:"verified"`
}

func (r *Repository) ActiveTripsCount(ctx context.Context) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM trips
		WHERE status IN ('searching','driver_assigned','driver_arriving','in_progress')
	`).Scan(&n)
	return n, err
}

func (r *Repository) ListActiveTrips(ctx context.Context) ([]ActiveTripItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.status::text, t.ride_type::text, t.estimated_fare,
		       u.phone,
		       COALESCE(vs.legal_name, op.full_name, ''),
		       COALESCE(vs.plate, ''),
		       ST_Y(t.pickup::geometry), ST_X(t.pickup::geometry),
		       op.last_lat, op.last_lng,
		       t.created_at, t.assigned_at
		FROM trips t
		JOIN users u ON u.id = t.passenger_id
		LEFT JOIN operator_profiles op ON op.id = t.operator_profile_id
		LEFT JOIN verified_operator_snapshots vs ON vs.operator_profile_id = op.id
		WHERE t.status IN ('searching','driver_assigned','driver_arriving','in_progress')
		ORDER BY t.updated_at DESC
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ActiveTripItem
	for rows.Next() {
		var item ActiveTripItem
		var id uuid.UUID
		var fare *int64
		var created time.Time
		var assigned *time.Time
		if err := rows.Scan(
			&id, &item.Status, &item.RideType, &fare,
			&item.PassengerPhone, &item.DriverName, &item.DriverPlate,
			&item.PickupLat, &item.PickupLng,
			&item.DriverLat, &item.DriverLng,
			&created, &assigned,
		); err != nil {
			return nil, err
		}
		item.ID = id.String()
		item.EstimatedFare = fare
		item.CreatedAt = created.UTC().Format(time.RFC3339)
		if assigned != nil {
			item.AssignedAt = assigned.UTC().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListOperators(ctx context.Context, status string) ([]OperatorListItem, error) {
	query := `
		SELECT op.id, u.phone,
		       COALESCE(NULLIF(op.full_name, ''), vs.legal_name, ''),
		       op.ride_type::text, op.status::text,
		       COALESCE(vs.plate, bd.plate, cd.plate, ''),
		       op.wallet_balance, op.wallet_min_balance,
		       EXISTS(SELECT 1 FROM verified_operator_snapshots vs2 WHERE vs2.operator_profile_id = op.id)
		FROM operator_profiles op
		JOIN users u ON u.id = op.user_id
		LEFT JOIN verified_operator_snapshots vs ON vs.operator_profile_id = op.id
		LEFT JOIN boda_details bd ON bd.operator_profile_id = op.id
		LEFT JOIN car_details cd ON cd.operator_profile_id = op.id
	`
	args := []any{}
	if status != "" {
		query += ` WHERE op.status = $1::operator_status`
		args = append(args, status)
	}
	query += ` ORDER BY op.updated_at DESC LIMIT 200`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OperatorListItem
	for rows.Next() {
		var item OperatorListItem
		var id uuid.UUID
		if err := rows.Scan(
			&id, &item.Phone, &item.Name, &item.RideType, &item.Status,
			&item.Plate, &item.WalletBalance, &item.WalletMin, &item.Verified,
		); err != nil {
			return nil, err
		}
		item.ID = id.String()
		items = append(items, item)
	}
	return items, rows.Err()
}
