package tailoring

// AllStatuses lists the seven schema enum values in workflow order.
var AllStatuses = []string{
	StatusSubmitted, StatusContacted, StatusConsultationScheduled,
	StatusMeasurementConfirmed, StatusInProgress, StatusCompleted, StatusCancelled,
}

// ValidStatus reports whether s is one of the seven schema statuses.
func ValidStatus(s string) bool {
	for _, v := range AllStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// MeasurementsLocked reports whether measurements can no longer be edited.
// Accepted limitation: there is no reopen flow yet.
func MeasurementsLocked(status string) bool {
	switch status {
	case StatusMeasurementConfirmed, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	}
	return false
}

// NextStatusAllowed implements the admin status endpoint transition table.
// StatusMeasurementConfirmed is never a valid target here: it is only reachable
// through the explicit confirm-measurements action. Same-status is reported as
// allowed by callers (no-op), not by this function.
func NextStatusAllowed(from, to string) bool {
	switch to {
	case StatusContacted:
		return from == StatusSubmitted
	case StatusConsultationScheduled:
		return from == StatusContacted
	case StatusInProgress:
		return from == StatusMeasurementConfirmed
	case StatusCompleted:
		return from == StatusInProgress
	case StatusCancelled:
		return from != StatusCompleted && from != StatusCancelled
	}
	return false
}

// CanConfirmMeasurements reports whether the status allows confirming measurements.
func CanConfirmMeasurements(status string) bool {
	return status == StatusContacted || status == StatusConsultationScheduled
}

// HasAllCMMeasurements reports whether all seven _cm measurements are present.
func HasAllCMMeasurements(m *Measurements) bool {
	return m != nil && m.NeckCM != nil && m.BustCM != nil && m.WaistCM != nil &&
		m.HipsCM != nil && m.ShoulderCM != nil && m.SleeveCM != nil && m.GarmentLengthCM != nil
}

// AllowedActions is computed by the server for UI display only; every
// operation is re-validated when executed.
func AllowedActions(r TailoringRequest) []string {
	actions := []string{"update"}
	if !MeasurementsLocked(r.Status) {
		actions = append(actions, "update_measurements")
	}
	switch r.Status {
	case StatusSubmitted:
		actions = append(actions, "contact")
	case StatusContacted:
		actions = append(actions, "schedule")
	case StatusMeasurementConfirmed:
		actions = append(actions, "start")
	case StatusInProgress:
		actions = append(actions, "complete")
	}
	if CanConfirmMeasurements(r.Status) && HasAllCMMeasurements(r.Measurements) {
		actions = append(actions, "confirm_measurements")
	}
	if r.Status != StatusCompleted && r.Status != StatusCancelled {
		actions = append(actions, "cancel")
	}
	return actions
}
