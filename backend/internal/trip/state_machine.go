package trip

// Status represents the trip lifecycle state stored in the database.
type Status string

const (
	StatusRequested      Status = "requested"
	StatusSearching      Status = "searching"
	StatusDriverAssigned Status = "driver_assigned"
	StatusDriverArriving Status = "driver_arriving"
	StatusInProgress     Status = "in_progress"
	StatusCompleted      Status = "completed"
	StatusCancelled      Status = "cancelled"
	StatusExpired        Status = "expired"
	StatusDisputed       Status = "disputed"
)

// RideType matches PostgreSQL ride_type enum.
type RideType string

const (
	RideBoda      RideType = "boda"
	RideCarBasic  RideType = "car_basic"
	RideCarXL     RideType = "car_xl"
)

// TransitionError is returned when a status change is not allowed.
type TransitionError struct {
	From Status
	To   Status
}

func (e TransitionError) Error() string {
	return "invalid trip transition: " + string(e.From) + " -> " + string(e.To)
}

// allowedTransitions defines the trip state machine.
var allowedTransitions = map[Status][]Status{
	StatusRequested:      {StatusSearching, StatusCancelled},
	StatusSearching:      {StatusDriverAssigned, StatusExpired, StatusCancelled},
	StatusDriverAssigned: {StatusDriverArriving, StatusSearching, StatusCancelled, StatusExpired},
	StatusDriverArriving: {StatusInProgress, StatusCancelled},
	StatusInProgress:     {StatusCompleted, StatusCancelled, StatusDisputed},
	StatusCompleted:      {StatusDisputed},
	StatusCancelled:      {},
	StatusExpired:        {},
	StatusDisputed:       {StatusCompleted, StatusCancelled},
}

// CanTransition reports whether from -> to is allowed.
func CanTransition(from, to Status) bool {
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, s := range next {
		if s == to {
			return true
		}
	}
	return false
}

// MustTransition returns nil if the transition is valid.
func MustTransition(from, to Status) error {
	if !CanTransition(from, to) {
		return TransitionError{From: from, To: to}
	}
	return nil
}

// IsActive reports whether the trip is in a non-terminal active state.
func IsActive(s Status) bool {
	switch s {
	case StatusSearching, StatusDriverAssigned, StatusDriverArriving, StatusInProgress:
		return true
	default:
		return false
	}
}
