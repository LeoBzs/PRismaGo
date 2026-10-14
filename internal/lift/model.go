// Package lift is the "lift log" feature: recording sets (exercise, weight,
// reps) that the dashboards and PR tracking are computed from.
//
// Layout of every feature package:
//
//	model.go      domain types + domain errors + pure domain logic
//	repository.go storage interface (+ in-memory impl for now)
//	service.go    use cases / business rules, depends only on the interface
//	handler.go    HTTP adapter: decode -> call service -> encode
package lift

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("lift not found")
	ErrUserNotFound = errors.New("user not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Lift struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Exercise    string    `json:"exercise"`
	WeightKg    float64   `json:"weight_kg"`
	Reps        int       `json:"reps"`
	PerformedAt time.Time `json:"performed_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// Estimated1RM uses the Epley formula. Pure domain logic lives on the model
// so the service, analytics and tests can all reuse it.
func (l Lift) Estimated1RM() float64 {
	if l.Reps <= 1 {
		return l.WeightKg
	}
	return l.WeightKg * (1 + float64(l.Reps)/30)
}
