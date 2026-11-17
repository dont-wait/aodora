package order

const (
	StatusConfirmed         = "confirmed"
	StatusShipping          = "shipping"
	StatusCancelled         = "cancelled"
	PaymentPaid             = "paid"
	PaymentRefunded         = "refunded"
	InventoryRestocked      = "restocked"
	InventoryAwaitingReturn = "awaiting_return"
	InventoryNotApplicable  = "not_applicable"
)

func NextStatusAllowed(paymentMethod, from, to string) bool {
	if to != StatusConfirmed && to != StatusShipping && to != StatusDelivered {
		return false
	}
	if from == to {
		return true
	}
	if from == StatusPending {
		return to == StatusConfirmed
	}
	if paymentMethod == PaymentMethodCashAtCounter {
		return from == StatusConfirmed && to == StatusDelivered
	}
	return (from == StatusConfirmed && to == StatusShipping) || (from == StatusShipping && to == StatusDelivered)
}

func NextPaymentAllowed(from, to string) bool {
	if to != PaymentPaid && to != PaymentRefunded {
		return false
	}
	return from == to || (from == PaymentUnpaid && to == PaymentPaid) || (from == PaymentPaid && to == PaymentRefunded)
}
