package clm

import "testing"

func TestCalibrate_CorrectsFactor(t *testing.T) {
	got := Calibrate(4000, 1000, 8.0)
	if got <= 4.0 || got >= 8.0 {
		t.Fatalf("factor should move toward 4.0 from 8.0, got %f", got)
	}
}

func TestCalibrate_ExactMatchStays(t *testing.T) {
	got := Calibrate(4000, 1000, 4.0)
	if got < 3.9 || got > 4.1 {
		t.Fatalf("factor should stay ~4.0, got %f", got)
	}
}
