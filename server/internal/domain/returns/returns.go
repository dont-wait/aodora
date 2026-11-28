// Package returns holds the return-request rules: who may ask, within which
// window, and which status may follow which.
package returns

import (
	"slices"
	"time"
)

// Window is how long after delivery a customer may ask to return items.
const Window = 72 * time.Hour

const (
	StatusRequested  = "requested"  // customer sent the request, waiting for the shop
	StatusApproved   = "approved"   // shop accepted, will collect the items
	StatusRejected   = "rejected"   // shop declined (reason in note)
	StatusCollecting = "collecting" // shipper/shop is collecting the items
	StatusCollected  = "collected"  // items are back at the shop
	StatusRefunded   = "refunded"   // money returned to the customer
	StatusCancelled  = "cancelled"  // customer withdrew the request before review
)

// Statuses in lifecycle order (used for filters and display).
var Statuses = []string{StatusRequested, StatusApproved, StatusCollecting, StatusCollected, StatusRefunded, StatusRejected, StatusCancelled}

var adminNext = map[string][]string{
	StatusRequested:  {StatusApproved, StatusRejected},
	StatusApproved:   {StatusCollecting, StatusRejected},
	StatusCollecting: {StatusCollected},
	StatusCollected:  {StatusRefunded, StatusRejected},
}

// AdminNext lists the statuses an admin may move a request to.
func AdminNext(from string) []string { return append([]string(nil), adminNext[from]...) }

// AdminCan reports whether an admin may move a request from one status to another.
func AdminCan(from, to string) bool { return slices.Contains(adminNext[from], to) }

// CustomerCanCancel: only a request the shop has not looked at yet.
func CustomerCanCancel(status string) bool { return status == StatusRequested }

// Open requests block a second request for the same order.
func Open(status string) bool {
	switch status {
	case StatusRequested, StatusApproved, StatusCollecting, StatusCollected:
		return true
	}
	return false
}

const (
	ReasonWrongSize      = "wrong_size"
	ReasonDefect         = "defect"
	ReasonNotAsDescribed = "not_as_described"
	ReasonWrongItem      = "wrong_item"
	ReasonChangedMind    = "changed_mind"
	ReasonOther          = "other"
)

var Reasons = []string{ReasonWrongSize, ReasonDefect, ReasonNotAsDescribed, ReasonWrongItem, ReasonChangedMind, ReasonOther}

func ValidReason(r string) bool { return slices.Contains(Reasons, r) }

// Deadline is the last moment a return may be requested for an order
// delivered at deliveredAt.
func Deadline(deliveredAt time.Time) time.Time { return deliveredAt.Add(Window) }

const (
	EvidenceImage = "image"
	EvidenceVideo = "video"
	MaxEvidence   = 6
)

type Evidence struct {
	URL         string `json:"url" bson:"url"`
	Kind        string `json:"kind" bson:"kind"`
	ContentType string `json:"content_type" bson:"content_type"`
}

type Item struct {
	ProductID   string `json:"product_id" bson:"product_id"`
	ProductName string `json:"product_name" bson:"product_name"`
	SKU         string `json:"sku" bson:"sku"`
	Size        string `json:"size,omitempty" bson:"size,omitempty"`
	Color       string `json:"color,omitempty" bson:"color,omitempty"`
	Quantity    int64  `json:"quantity" bson:"quantity"`
	UnitPrice   int64  `json:"unit_price" bson:"unit_price"`
}

type Event struct {
	Status string    `json:"status" bson:"status"`
	At     time.Time `json:"at" bson:"at"`
	By     string    `json:"by" bson:"by"` // "customer" or "admin"
	Note   string    `json:"note,omitempty" bson:"note,omitempty"`
}

type Request struct {
	ID           string     `json:"id" bson:"id"`
	OrderID      string     `json:"order_id" bson:"order_id"`
	OrderNumber  string     `json:"order_number" bson:"order_number"`
	CustomerID   string     `json:"customer_id" bson:"customer_id"`
	CustomerName string     `json:"customer_name,omitempty" bson:"customer_name,omitempty"`
	Phone        string     `json:"phone,omitempty" bson:"phone,omitempty"`
	Items        []Item     `json:"items" bson:"items"`
	Reason       string     `json:"reason" bson:"reason"`
	Note         string     `json:"note,omitempty" bson:"note,omitempty"`
	Evidence     []Evidence `json:"evidence" bson:"evidence"`
	Status       string     `json:"status" bson:"status"`
	RefundAmount int64      `json:"refund_amount" bson:"refund_amount"`
	Restocked    bool       `json:"restocked" bson:"restocked"`
	AdminNote    string     `json:"admin_note,omitempty" bson:"admin_note,omitempty"`
	History      []Event    `json:"history" bson:"history"`
	DeliveredAt  time.Time  `json:"delivered_at" bson:"delivered_at"`
	CreatedAt    time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" bson:"updated_at"`
}
