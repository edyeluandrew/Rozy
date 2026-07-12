package trip

import "math"

// FareRule holds pricing config for one ride category in a city.
type FareRule struct {
	BaseFare           int64
	PerKmRate          int64
	MinFare            int64
	RozyFeeFixed       int64
	RoundTo            int64
	RoadFactorFallback float64
	MinBillableKm      float64
}

// CalculateFare computes passenger fare from distance in kilometers.
// Formula: max(min_fare, base + km * per_km), rounded to nearest RoundTo.
func CalculateFare(rule FareRule, distanceKm float64) int64 {
	if distanceKm < rule.MinBillableKm {
		distanceKm = rule.MinBillableKm
	}

	raw := float64(rule.BaseFare) + distanceKm*float64(rule.PerKmRate)
	fare := int64(math.Max(float64(rule.MinFare), raw))

	if rule.RoundTo > 0 {
		fare = roundToNearest(fare, rule.RoundTo)
	}
	return fare
}

// EstimateDistanceKm returns routed km or haversine * road factor fallback.
func EstimateDistanceKm(routedKm float64, haversineKm float64, roadFactor float64) float64 {
	if routedKm > 0 {
		return routedKm
	}
	if roadFactor <= 0 {
		roadFactor = 1.3
	}
	return haversineKm * roadFactor
}

func roundToNearest(value, unit int64) int64 {
	if unit <= 0 {
		return value
	}
	remainder := value % unit
	if remainder >= unit/2 {
		return value + (unit - remainder)
	}
	return value - remainder
}

// HaversineKm returns great-circle distance in km between two WGS84 points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}
