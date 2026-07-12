package trip

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

// DriverSnapshot is shown to the passenger while a driver is assigned.
type DriverSnapshot struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Plate               string  `json:"plate"`
	RideType            string  `json:"ride_type"`
	Lat                 float64 `json:"lat"`
	Lng                 float64 `json:"lng"`
	LocationUpdatedAt   string  `json:"location_updated_at,omitempty"`
}

// Tracking adds live driver ETA for passenger maps.
type Tracking struct {
	Driver           *DriverSnapshot `json:"driver,omitempty"`
	DriverDistanceKm *float64        `json:"driver_distance_km,omitempty"`
	DriverEtaMinutes *int            `json:"driver_eta_minutes,omitempty"`
}

func (r *Repository) enrichPassengerTrip(ctx context.Context, t *Trip) error {
	if t == nil {
		return nil
	}
	tripID, err := uuid.Parse(t.ID)
	if err != nil {
		return err
	}

	var opID *uuid.UUID
	var lastLat, lastLng *float64
	var lastAt *time.Time
	var legalName, plate *string
	var opRideType *string

	err = r.pool.QueryRow(ctx, `
		SELECT t.operator_profile_id,
		       op.last_lat, op.last_lng, op.last_location_at,
		       vs.legal_name, vs.plate, op.ride_type::text
		FROM trips t
		LEFT JOIN operator_profiles op ON op.id = t.operator_profile_id
		LEFT JOIN verified_operator_snapshots vs ON vs.operator_profile_id = op.id
		WHERE t.id = $1
	`, tripID).Scan(&opID, &lastLat, &lastLng, &lastAt, &legalName, &plate, &opRideType)
	if err != nil {
		return err
	}
	if opID == nil {
		return nil
	}

	tracking := buildTracking(t, opID.String(), legalName, plate, opRideType, lastLat, lastLng, lastAt)
	t.Driver = tracking.Driver
	t.DriverDistanceKm = tracking.DriverDistanceKm
	t.DriverEtaMinutes = tracking.DriverEtaMinutes
	return nil
}

func buildTracking(
	t *Trip,
	opID string,
	legalName, plate, opRideType *string,
	lastLat, lastLng *float64,
	lastAt *time.Time,
) Tracking {
	if lastLat == nil || lastLng == nil {
		return Tracking{}
	}

	name := "Rozy Driver"
	if legalName != nil && *legalName != "" {
		name = *legalName
	}
	plateStr := ""
	if plate != nil {
		plateStr = *plate
	}
	rideType := t.RideType
	if opRideType != nil && *opRideType != "" {
		rideType = *opRideType
	}

	driver := &DriverSnapshot{
		ID:       opID,
		Name:     name,
		Plate:    plateStr,
		RideType: rideType,
		Lat:      *lastLat,
		Lng:      *lastLng,
	}
	if lastAt != nil {
		driver.LocationUpdatedAt = lastAt.UTC().Format(time.RFC3339)
	}

	var targetLat, targetLng float64
	includeTracking := false

	switch Status(t.Status) {
	case StatusDriverAssigned, StatusDriverArriving:
		if t.ArrivedAt == "" {
			targetLat, targetLng = t.PickupLat, t.PickupLng
			includeTracking = true
		}
	case StatusInProgress:
		targetLat, targetLng = t.DestLat, t.DestLng
		includeTracking = true
	}

	if !includeTracking {
		return Tracking{Driver: driver}
	}

	dist := HaversineKm(*lastLat, *lastLng, targetLat, targetLng)
	dist = math.Round(dist*10) / 10
	eta := etaMinutes(dist, rideType)

	return Tracking{
		Driver:           driver,
		DriverDistanceKm: &dist,
		DriverEtaMinutes: &eta,
	}
}

func etaMinutes(distanceKm float64, rideType string) int {
	speedKmh := 22.0 // boda in Mbarara traffic
	switch rideType {
	case "car_basic", "car_xl":
		speedKmh = 20.0
	}
	if distanceKm <= 0.05 {
		return 1
	}
	mins := math.Ceil((distanceKm / speedKmh) * 60)
	if mins < 1 {
		return 1
	}
	return int(mins)
}
