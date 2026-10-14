// Package user holds user profile data. Body measurements are always stored
// in metric; UnitSystem is only the user's display/input preference.
package user

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("user not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrEmailTaken   = errors.New("email already registered")
)

type UnitSystem string

const (
	Metric   UnitSystem = "metric"   // cm, kg
	Imperial UnitSystem = "imperial" // in, lb
)

const (
	cmPerInch = 2.54
	kgPerLb   = 0.45359237
)

// Plausible ranges, in metric. Imperial input is converted before checking,
// so both systems share the same bounds.
const (
	minHeightCm = 100.0
	maxHeightCm = 250.0
	minWeightKg = 30.0
	maxWeightKg = 300.0
)

func (u UnitSystem) Valid() bool {
	return u == Metric || u == Imperial
}

type Sex string

const (
	Male   Sex = "male"
	Female Sex = "female"
)

func (s Sex) Valid() bool {
	return s == Male || s == Female
}

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	Sex          Sex        `json:"sex"`
	Units        UnitSystem `json:"units"`
	HeightCm     float64    `json:"height_cm"`
	BodyWeightKg float64    `json:"body_weight_kg"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Measurements is body data as the client sent it, in Units.
type Measurements struct {
	Units  UnitSystem `json:"units"`
	Height float64    `json:"height"` // cm or in
	Weight float64    `json:"weight"` // kg or lb
}

// ToMetric validates m and returns height (cm) and weight (kg).
func (m Measurements) ToMetric() (heightCm, weightKg float64, err error) {
	switch m.Units {
	case Metric:
		heightCm, weightKg = m.Height, m.Weight
	case Imperial:
		heightCm, weightKg = m.Height*cmPerInch, m.Weight*kgPerLb
	default:
		return 0, 0, fmt.Errorf("%w: units must be %q or %q", ErrInvalidInput, Metric, Imperial)
	}

	hUnit, wUnit := m.Units.labels()
	if heightCm < minHeightCm || heightCm > maxHeightCm {
		lo, hi := m.Units.fromHeightCm(minHeightCm), m.Units.fromHeightCm(maxHeightCm)
		return 0, 0, fmt.Errorf("%w: height must be between %.0f and %.0f %s", ErrInvalidInput, lo, hi, hUnit)
	}
	if weightKg < minWeightKg || weightKg > maxWeightKg {
		lo, hi := m.Units.fromWeightKg(minWeightKg), m.Units.fromWeightKg(maxWeightKg)
		return 0, 0, fmt.Errorf("%w: weight must be between %.0f and %.0f %s", ErrInvalidInput, lo, hi, wUnit)
	}
	return heightCm, weightKg, nil
}

// Validate checks a fully built User (canonical metric values).
func (u User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("%w: email is invalid", ErrInvalidInput)
	}
	_, _, err := Measurements{Units: Metric, Height: u.HeightCm, Weight: u.BodyWeightKg}.ToMetric()
	if err != nil {
		return err
	}
	if !u.Units.Valid() {
		return fmt.Errorf("%w: units must be %q or %q", ErrInvalidInput, Metric, Imperial)
	}
	if !u.Sex.Valid() {
		return fmt.Errorf("%w: sex must be %q or %q", ErrInvalidInput, Male, Female)
	}
	return nil
}

// IdealWeightKg uses the Devine formula: 50 kg (male) or 45.5 kg (female)
// plus 2.3 kg per inch over 5 ft. Devine is only defined from 5 ft up, so
// shorter users get the base value. Note it ignores muscle mass.
func (u User) IdealWeightKg() float64 {
	base := 50.0
	if u.Sex == Female {
		base = 45.5
	}
	inchesOver5ft := u.HeightCm/cmPerInch - 60
	if inchesOver5ft < 0 {
		inchesOver5ft = 0
	}
	return base + 2.3*inchesOver5ft
}

// IdealWeight returns IdealWeightKg in the user's preferred units.
func (u User) IdealWeight() float64 {
	return u.Units.fromWeightKg(u.IdealWeightKg())
}

// Display returns the user's measurements in their preferred unit system.
func (u User) Display() Measurements {
	return Measurements{
		Units:  u.Units,
		Height: u.Units.fromHeightCm(u.HeightCm),
		Weight: u.Units.fromWeightKg(u.BodyWeightKg),
	}
}

func (u UnitSystem) labels() (height, weight string) {
	if u == Imperial {
		return "in", "lb"
	}
	return "cm", "kg"
}

func (u UnitSystem) fromHeightCm(cm float64) float64 {
	if u == Imperial {
		return cm / cmPerInch
	}
	return cm
}

func (u UnitSystem) fromWeightKg(kg float64) float64 {
	if u == Imperial {
		return kg / kgPerLb
	}
	return kg
}
