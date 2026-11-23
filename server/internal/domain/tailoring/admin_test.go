package tailoring

import "testing"

func TestNextStatusAllowed(t *testing.T) {
	ok := [][2]string{
		{StatusSubmitted, StatusContacted},
		{StatusContacted, StatusConsultationScheduled},
		{StatusMeasurementConfirmed, StatusInProgress},
		{StatusInProgress, StatusCompleted},
		{StatusSubmitted, StatusCancelled}, {StatusContacted, StatusCancelled},
		{StatusConsultationScheduled, StatusCancelled}, {StatusMeasurementConfirmed, StatusCancelled},
		{StatusInProgress, StatusCancelled},
	}
	for _, c := range ok {
		if !NextStatusAllowed(c[0], c[1]) {
			t.Errorf("%s -> %s should be allowed", c[0], c[1])
		}
	}
	bad := [][2]string{
		{StatusSubmitted, StatusConsultationScheduled},  // skip
		{StatusContacted, StatusSubmitted},              // backwards
		{StatusConsultationScheduled, StatusInProgress}, // must confirm first
		{StatusConsultationScheduled, StatusMeasurementConfirmed},
		{StatusSubmitted, StatusMeasurementConfirmed},
		{StatusMeasurementConfirmed, StatusCompleted},
		{StatusCompleted, StatusCancelled}, {StatusCancelled, StatusContacted},
		{StatusCompleted, StatusInProgress},
	}
	for _, c := range bad {
		if NextStatusAllowed(c[0], c[1]) {
			t.Errorf("%s -> %s should be rejected", c[0], c[1])
		}
	}
}

func TestMeasurementsLockedAndActions(t *testing.T) {
	for _, s := range AllStatuses {
		want := s == StatusMeasurementConfirmed || s == StatusInProgress || s == StatusCompleted || s == StatusCancelled
		if MeasurementsLocked(s) != want {
			t.Errorf("MeasurementsLocked(%s) != %v", s, want)
		}
	}
	f := 50.0
	full := &Measurements{NeckCM: &f, BustCM: &f, WaistCM: &f, HipsCM: &f, ShoulderCM: &f, SleeveCM: &f, GarmentLengthCM: &f}
	has := func(a []string, v string) bool {
		for _, x := range a {
			if x == v {
				return true
			}
		}
		return false
	}
	if !has(AllowedActions(TailoringRequest{Status: StatusContacted, Measurements: full}), "confirm_measurements") {
		t.Error("contacted + full should offer confirm_measurements")
	}
	if has(AllowedActions(TailoringRequest{Status: StatusContacted}), "confirm_measurements") {
		t.Error("incomplete measurements must not offer confirm")
	}
	if has(AllowedActions(TailoringRequest{Status: StatusCompleted}), "cancel") {
		t.Error("completed must not offer cancel")
	}
	if has(AllowedActions(TailoringRequest{Status: StatusInProgress}), "update_measurements") {
		t.Error("locked must not offer update_measurements")
	}
}
