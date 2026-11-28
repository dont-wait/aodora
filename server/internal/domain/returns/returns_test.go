package returns

import (
	"testing"
	"time"
)

func TestAdminTransitions(t *testing.T) {
	ok := [][2]string{{StatusRequested, StatusApproved}, {StatusRequested, StatusRejected}, {StatusApproved, StatusCollecting}, {StatusCollecting, StatusCollected}, {StatusCollected, StatusRefunded}, {StatusCollected, StatusRejected}}
	for _, c := range ok {
		if !AdminCan(c[0], c[1]) {
			t.Errorf("%s -> %s should be allowed", c[0], c[1])
		}
	}
	bad := [][2]string{{StatusRequested, StatusRefunded}, {StatusRequested, StatusCollecting}, {StatusRefunded, StatusRequested}, {StatusRejected, StatusApproved}, {StatusCollecting, StatusRefunded}, {StatusCancelled, StatusApproved}}
	for _, c := range bad {
		if AdminCan(c[0], c[1]) {
			t.Errorf("%s -> %s must be refused", c[0], c[1])
		}
	}
}

func TestOpenAndCancel(t *testing.T) {
	if !Open(StatusCollecting) || Open(StatusRejected) || Open(StatusRefunded) || Open(StatusCancelled) {
		t.Fatal("open statuses wrong")
	}
	if !CustomerCanCancel(StatusRequested) || CustomerCanCancel(StatusApproved) {
		t.Fatal("customer may cancel only before review")
	}
}

func TestDeadlineIsThreeDays(t *testing.T) {
	d := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	if got := Deadline(d); !got.Equal(time.Date(2026, 11, 28, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("deadline %s", got)
	}
}
