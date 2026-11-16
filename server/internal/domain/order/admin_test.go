package order

import "testing"

func TestAdminTransitionTables(t *testing.T) {
	statuses := []string{StatusPending, StatusConfirmed, StatusShipping, StatusDelivered, StatusCancelled, "unknown"}
	for _, method := range []string{PaymentMethodCOD, PaymentMethodCashAtCounter, "bank_transfer"} {
		for _, from := range statuses {
			for _, to := range statuses {
				want := false
				if to == StatusConfirmed || to == StatusShipping || to == StatusDelivered {
					want = from == to || (from == StatusPending && to == StatusConfirmed)
					if method == PaymentMethodCashAtCounter {
						want = want || (from == StatusConfirmed && to == StatusDelivered)
					} else {
						want = want || (from == StatusConfirmed && to == StatusShipping) || (from == StatusShipping && to == StatusDelivered)
					}
				}
				if got := NextStatusAllowed(method, from, to); got != want {
					t.Errorf("%s %s -> %s: %v, want %v", method, from, to, got, want)
				}
			}
		}
	}
	for _, from := range []string{PaymentUnpaid, PaymentPaid, PaymentRefunded, "unknown"} {
		for _, to := range []string{PaymentUnpaid, PaymentPaid, PaymentRefunded, "unknown"} {
			want := (to == PaymentPaid || to == PaymentRefunded) && (from == to || (from == PaymentUnpaid && to == PaymentPaid) || (from == PaymentPaid && to == PaymentRefunded))
			if got := NextPaymentAllowed(from, to); got != want {
				t.Errorf("payment %s -> %s: %v want %v", from, to, got, want)
			}
		}
	}
}
