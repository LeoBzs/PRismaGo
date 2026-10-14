package user

import (
	"errors"
	"math"
	"testing"
)

func TestMeasurementsToMetric(t *testing.T) {
	tests := []struct {
		name    string
		in      Measurements
		wantErr error
	}{
		{"metric valid", Measurements{Metric, 180, 85}, nil},
		{"imperial valid", Measurements{Imperial, 71, 187}, nil},
		{"imperial height sent as cm", Measurements{Imperial, 180, 187}, ErrInvalidInput},
		{"imperial weight too low", Measurements{Imperial, 71, 50}, ErrInvalidInput},
		{"metric weight sent as lb", Measurements{Metric, 180, 400}, ErrInvalidInput},
		{"unknown units", Measurements{"stone", 180, 85}, ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := tt.in.ToMetric()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestIdealWeight(t *testing.T) {
	tests := []struct {
		name   string
		u      User
		wantKg float64
	}{
		{"male 6ft", User{Sex: Male, HeightCm: 72 * cmPerInch}, 50 + 2.3*12},
		{"female 5ft6", User{Sex: Female, HeightCm: 66 * cmPerInch}, 45.5 + 2.3*6},
		{"under 5ft uses base", User{Sex: Female, HeightCm: 140}, 45.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.u.IdealWeightKg(); math.Abs(got-tt.wantKg) > 1e-9 {
				t.Errorf("IdealWeightKg = %v, want %v", got, tt.wantKg)
			}
		})
	}

	u := User{Sex: Male, Units: Imperial, HeightCm: 72 * cmPerInch}
	if got, want := u.IdealWeight(), (50+2.3*12)/kgPerLb; math.Abs(got-want) > 1e-9 {
		t.Errorf("IdealWeight imperial = %v, want %v lb", got, want)
	}
}

func TestImperialRoundTrip(t *testing.T) {
	h, w, err := Measurements{Imperial, 70, 200}.ToMetric()
	if err != nil {
		t.Fatal(err)
	}
	got := User{Units: Imperial, HeightCm: h, BodyWeightKg: w}.Display()
	if math.Abs(got.Height-70) > 1e-9 || math.Abs(got.Weight-200) > 1e-9 {
		t.Errorf("round trip = %+v, want 70in / 200lb", got)
	}
}
