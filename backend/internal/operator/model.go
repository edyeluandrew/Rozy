package operator

import (
	"errors"
	"fmt"
)

var (
	ErrAlreadyRegistered = errors.New("operator already registered")
	ErrNotDriver         = errors.New("user is not a driver account")
	ErrInvalidRideType   = errors.New("invalid ride type")
	ErrNotFound          = errors.New("operator profile not found")
)

type RideType string

const (
	RideBoda     RideType = "boda"
	RideCarBasic RideType = "car_basic"
	RideCarXL    RideType = "car_xl"
)

type OperatorType string

const (
	TypeBoda OperatorType = "boda"
	TypeCar  OperatorType = "car"
)

type Profile struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	OperatorType  OperatorType `json:"operator_type"`
	RideType      RideType     `json:"ride_type"`
	Status        string       `json:"status"`
	WalletBalance int64        `json:"wallet_balance"`
	WalletMin     int64        `json:"wallet_min_balance"`
}

func ParseRideType(raw string) (OperatorType, RideType, error) {
	switch RideType(raw) {
	case RideBoda:
		return TypeBoda, RideBoda, nil
	case RideCarBasic:
		return TypeCar, RideCarBasic, nil
	case RideCarXL:
		return TypeCar, RideCarXL, nil
	default:
		return "", "", fmt.Errorf("%w: %s", ErrInvalidRideType, raw)
	}
}
