package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

type fakeCatalog struct {
	products map[string]Product
}

func (c *fakeCatalog) SellableProducts(ctx context.Context, productIDs []string) (map[string]Product, error) {
	res := make(map[string]Product)
	for _, id := range productIDs {
		if p, ok := c.products[id]; ok {
			res[id] = p
		}
	}
	return res, nil
}

type fakeTx struct {
	store         *fakeStore
	stockSnapshot map[string]Product
}

func (tx *fakeTx) SellableProducts(ctx context.Context, productIDs []string) (map[string]Product, error) {
	return tx.store.getProducts(productIDs), nil
}

func (tx *fakeTx) DecrementStock(ctx context.Context, lines []StockLine) error {
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()

	for _, line := range lines {
		prod, ok := tx.store.products[line.ProductID]
		if !ok {
			return ErrOutOfStock
		}
		if line.HasVariants {
			found := false
			for i := range prod.Variants {
				if prod.Variants[i].SKU == line.SKU {
					found = true
					if prod.Variants[i].Stock < line.Quantity {
						return ErrOutOfStock
					}
					prod.Variants[i].Stock -= line.Quantity
					break
				}
			}
			if !found {
				return ErrOutOfStock
			}
		}
		if prod.Stock < line.Quantity {
			return ErrOutOfStock
		}
		prod.Stock -= line.Quantity
		tx.store.products[line.ProductID] = prod
	}
	return nil
}

func (tx *fakeTx) InsertOrder(ctx context.Context, o domain.Order) (domain.Order, error) {
	if tx.store.insertOrderHook != nil {
		return tx.store.insertOrderHook(ctx, o)
	}
	tx.store.mu.Lock()
	defer tx.store.mu.Unlock()

	key := o.Checkout.Scope + "|" + o.Checkout.RequestID
	if _, exists := tx.store.ordersByAttempt[key]; exists {
		return domain.Order{}, ErrDuplicateAttempt
	}
	for _, existing := range tx.store.ordersByID {
		if existing.OrderNumber == o.OrderNumber {
			return domain.Order{}, ErrDuplicateOrderNumber
		}
	}

	if o.ID == "" {
		o.ID = fmt.Sprintf("6900000000000000000000%02d", len(tx.store.ordersByID)+1)
	}
	tx.store.ordersByID[o.ID] = o
	tx.store.ordersByAttempt[key] = o
	return o, nil
}

type fakeStore struct {
	mu              sync.Mutex
	products        map[string]Product
	ordersByID      map[string]domain.Order
	ordersByAttempt map[string]domain.Order

	runCheckoutHook func(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error
	insertOrderHook func(ctx context.Context, o domain.Order) (domain.Order, error)
	findAttemptHook func(ctx context.Context, scope, requestID string) (*domain.Order, error)
	findByIDHook    func(ctx context.Context, id string) (*domain.Order, error)
}

func newFakeStore(products map[string]Product) *fakeStore {
	prodCopy := make(map[string]Product)
	for k, v := range products {
		var vars []Variant
		vars = append(vars, v.Variants...)
		v.Variants = vars
		prodCopy[k] = v
	}
	return &fakeStore{
		products:        prodCopy,
		ordersByID:      make(map[string]domain.Order),
		ordersByAttempt: make(map[string]domain.Order),
	}
}

func (s *fakeStore) cloneOrders() (map[string]domain.Order, map[string]domain.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	orders := make(map[string]domain.Order)
	for k, v := range s.ordersByID {
		orders[k] = v
	}
	attempts := make(map[string]domain.Order)
	for k, v := range s.ordersByAttempt {
		attempts[k] = v
	}
	return orders, attempts
}

func (s *fakeStore) restoreOrders(orders, attempts map[string]domain.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ordersByID = orders
	s.ordersByAttempt = attempts
}

func (s *fakeStore) cloneProducts() map[string]Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make(map[string]Product)
	for k, v := range s.products {
		var vars []Variant
		vars = append(vars, v.Variants...)
		v.Variants = vars
		res[k] = v
	}
	return res
}

func (s *fakeStore) restoreProducts(snapshot map[string]Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products = snapshot
}

func (s *fakeStore) getProducts(productIDs []string) map[string]Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make(map[string]Product)
	for _, id := range productIDs {
		if p, ok := s.products[id]; ok {
			var vars []Variant
			vars = append(vars, p.Variants...)
			p.Variants = vars
			res[id] = p
		}
	}
	return res
}

func (s *fakeStore) RunCheckout(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error {
	if s.runCheckoutHook != nil {
		return s.runCheckoutHook(ctx, fn)
	}

	snapshot := s.cloneProducts()
	ordersSnap, attemptsSnap := s.cloneOrders()
	tx := &fakeTx{store: s, stockSnapshot: snapshot}
	err := fn(ctx, tx)
	if err != nil {
		s.restoreProducts(snapshot)
		s.restoreOrders(ordersSnap, attemptsSnap)
		return err
	}
	return nil
}

func (s *fakeStore) FindByAttempt(ctx context.Context, scope, requestID string) (*domain.Order, error) {
	if s.findAttemptHook != nil {
		return s.findAttemptHook(ctx, scope, requestID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scope + "|" + requestID
	if o, ok := s.ordersByAttempt[key]; ok {
		return &o, nil
	}
	return nil, nil
}

func (s *fakeStore) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	if s.findByIDHook != nil {
		return s.findByIDHook(ctx, id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if o, ok := s.ordersByID[id]; ok {
		return &o, nil
	}
	return nil, nil
}

func (s *fakeStore) ListByCustomer(ctx context.Context, customerID, status string, page, pageSize int64) ([]domain.Order, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []domain.Order
	for _, o := range s.ordersByID {
		if o.CustomerID == customerID && (status == "" || o.Status == status) {
			all = append(all, o)
		}
	}
	total := int64(len(all))
	start := (page - 1) * pageSize
	if start >= total {
		return nil, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func testSigner(t *testing.T) *domain.QuoteSigner {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i + 1)
	}
	s, err := domain.NewQuoteSigner(secret)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestQuote(t *testing.T) {
	p1Price, _ := domain.ParseMoney("199999.5")
	p2Price, _ := domain.ParseMoney("100000")
	products := map[string]Product{
		"680000000000000000000001": {
			ID:    "680000000000000000000001",
			Name:  "Áo dài lụa",
			SKU:   "LUA-DO",
			Price: p1Price,
			Stock: 10,
			Variants: []Variant{
				{SKU: "LUA-DO-M", Color: "Đỏ", Size: "M", Stock: 5},
				{SKU: "LUA-DO-L", Color: "Đỏ", Size: "L", Stock: 5},
			},
		},
		"680000000000000000000002": {
			ID:    "680000000000000000000002",
			Name:  "Khăn lụa",
			SKU:   "KHAN-LUA",
			Price: p2Price,
			Stock: 5,
		},
	}

	cat := &fakeCatalog{products: products}
	store := newFakeStore(products)
	signer := testSigner(t)
	codFee, _ := domain.ParseMoney("30000")
	nowTime := time.Unix(1700000000, 0)
	nowFunc := func() time.Time { return nowTime }

	svc := NewService(cat, store, signer, 15*time.Minute, &codFee, nowFunc, nil)

	ctx := context.Background()

	// 1. Cash at counter: shipping fee is 0, total = subtotal
	counterInput := domain.CheckoutInput{
		PaymentMethod: domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
		},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 3},
		},
	}
	q1, err := svc.Quote(ctx, counterInput)
	if err != nil {
		t.Fatalf("unexpected error for counter quote: %v", err)
	}
	if q1.ShippingFee.String() != "0" {
		t.Errorf("shipping fee = %s, want 0", q1.ShippingFee.String())
	}
	// 199999.5 * 3 = 599998.5
	if q1.Subtotal.String() != "599998.5" || q1.Total.String() != "599998.5" {
		t.Errorf("subtotal = %s, total = %s, want 599998.5", q1.Subtotal.String(), q1.Total.String())
	}

	// Verify the returned token is valid and decrypts to same info
	verified, err := signer.Verify(q1.QuoteToken, nowTime)
	if err != nil {
		t.Fatalf("quote token verification failed: %v", err)
	}
	if verified.Total != "599998.5" || len(verified.Items) != 1 {
		t.Errorf("unexpected verified quote payload: %+v", verified)
	}

	// 2. COD has configured shipping fee: 599998.5 + 30000 = 629998.5
	codInput := domain.CheckoutInput{
		PaymentMethod: domain.PaymentMethodCOD,
		ShippingAddress: domain.ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
			Line1:     "12 Nguyễn Trãi",
			City:      "TP. Hồ Chí Minh",
		},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 3},
		},
	}
	q2, err := svc.Quote(ctx, codInput)
	if err != nil {
		t.Fatalf("unexpected error for COD quote: %v", err)
	}
	if q2.ShippingFee.String() != "30000" {
		t.Errorf("COD shipping fee = %s, want 30000", q2.ShippingFee.String())
	}
	if q2.Total.String() != "629998.5" {
		t.Errorf("COD total = %s, want 629998.5", q2.Total.String())
	}

	// 3. COD with fee unconfigured (nil) => ErrShippingUnavailable
	svcNofee := NewService(cat, store, signer, 15*time.Minute, nil, nowFunc, nil)
	if _, err := svcNofee.Quote(ctx, codInput); !errors.Is(err, ErrShippingUnavailable) {
		t.Errorf("expected ErrShippingUnavailable, got %v", err)
	}

	// 4. Signer is nil => ErrOrderUnavailable
	svcNoSigner := NewService(cat, store, nil, 15*time.Minute, &codFee, nowFunc, nil)
	if _, err := svcNoSigner.Quote(ctx, counterInput); !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected ErrOrderUnavailable, got %v", err)
	}

	// 5. Product missing or wrong SKU reports InvalidField with original SourceIndex
	wrongProdInput := domain.CheckoutInput{
		PaymentMethod:   domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 1}, // index 0 (valid)
			{ProductID: "680000000000000000000099", SKU: "KHAN-LUA", Quantity: 1}, // index 1 (missing product)
		},
	}
	_, err = svc.Quote(ctx, wrongProdInput)
	var ifErr *domain.InvalidFieldError
	if !errors.As(err, &ifErr) || ifErr.Field != "items[1].product_id" {
		t.Errorf("expected error on items[1].product_id, got %v", err)
	}

	wrongSKUInput := domain.CheckoutInput{
		PaymentMethod:   domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000002", SKU: "KHAN-LUA", Quantity: 1},  // index 0 (valid)
			{ProductID: "680000000000000000000001", SKU: "WRONG-SKU", Quantity: 1}, // index 1 (invalid SKU)
		},
	}
	_, err = svc.Quote(ctx, wrongSKUInput)
	if !errors.As(err, &ifErr) || ifErr.Field != "items[1].sku" {
		t.Errorf("expected error on items[1].sku, got %v", err)
	}

	// 6. Out of stock (single variant has 5 stock, requesting 6)
	oosInput := domain.CheckoutInput{
		PaymentMethod:   domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 6},
		},
	}
	if _, err := svc.Quote(ctx, oosInput); !errors.Is(err, ErrOutOfStock) {
		t.Errorf("expected ErrOutOfStock, got %v", err)
	}

	// 7. Two SKUs of same product exceed total product stock (each 4, total 8, product stock 10 -> ok; each 6, total 12 > 10 -> oos)
	exceedProductStockInput := domain.CheckoutInput{
		PaymentMethod:   domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{Recipient: "A", Phone: "1"},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 5},
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-L", Quantity: 6}, // exceeds product stock 10
		},
	}
	if _, err := svc.Quote(ctx, exceedProductStockInput); !errors.Is(err, ErrOutOfStock) {
		t.Errorf("expected ErrOutOfStock for exceeding overall product stock, got %v", err)
	}
}

func TestCreate(t *testing.T) {
	p1Price, _ := domain.ParseMoney("500000")
	products := map[string]Product{
		"680000000000000000000001": {
			ID:    "680000000000000000000001",
			Name:  "Áo dài lụa",
			SKU:   "LUA-DO",
			Price: p1Price,
			Stock: 10,
			Variants: []Variant{
				{SKU: "LUA-DO-M", Color: "Đỏ", Size: "M", Stock: 5},
			},
		},
	}

	cat := &fakeCatalog{products: products}
	store := newFakeStore(products)
	signer := testSigner(t)
	codFee, _ := domain.ParseMoney("30000")
	nowTime := time.Unix(1700000000, 0)
	nowFunc := func() time.Time { return nowTime }
	orderNumCounter := 0
	orderNumFunc := func() string {
		orderNumCounter++
		return fmt.Sprintf("AD-261116-%04d", orderNumCounter)
	}

	svc := NewService(cat, store, signer, 15*time.Minute, &codFee, nowFunc, orderNumFunc)
	ctx := context.Background()

	// 1. Initial successful Quote
	checkoutInput := domain.CheckoutInput{
		PaymentMethod: domain.PaymentMethodCashAtCounter,
		ShippingAddress: domain.ShippingAddressInput{
			Recipient: "Nguyễn An",
			Phone:     "0900000001",
		},
		Items: []domain.OrderItemInput{
			{ProductID: "680000000000000000000001", SKU: "LUA-DO-M", Quantity: 2},
		},
	}
	quoteRes, err := svc.Quote(ctx, checkoutInput)
	if err != nil {
		t.Fatalf("Quote failed: %v", err)
	}
	checkoutInput.QuoteToken = quoteRes.QuoteToken

	guestPrincipal := Principal{GuestHash: "630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"}
	requestID := "ad577a65-cd42-46eb-ae8a-98b5f1186571"

	// 2. Create new order
	order, created, err := svc.Create(ctx, guestPrincipal, requestID, checkoutInput)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if !created {
		t.Errorf("expected created = true on first call")
	}
	if order.Status != domain.StatusPending || order.PaymentStatus != domain.PaymentUnpaid {
		t.Errorf("status = %s, paymentStatus = %s", order.Status, order.PaymentStatus)
	}
	if order.CustomerID != "" {
		t.Errorf("guest order should have empty CustomerID, got %s", order.CustomerID)
	}
	if order.GuestAccessHash != guestPrincipal.GuestHash {
		t.Errorf("guest access hash = %s, want %s", order.GuestAccessHash, guestPrincipal.GuestHash)
	}
	if order.Total.String() != "1000000" {
		t.Errorf("order total = %s, want 1000000", order.Total.String())
	}
	// Check stock was decremented from 5 to 3
	if store.products["680000000000000000000001"].Variants[0].Stock != 3 {
		t.Errorf("expected stock 3, got %d", store.products["680000000000000000000001"].Variants[0].Stock)
	}

	// 3. Replay with identical payload returns existing order with created = false
	// EVEN IF quote token has expired and product price has changed in catalog!
	nowTime = nowTime.Add(2 * time.Hour) // Advance time way past quote expiry
	newPrice, _ := domain.ParseMoney("999999")
	pCopy := store.products["680000000000000000000001"]
	pCopy.Price = newPrice
	store.products["680000000000000000000001"] = pCopy

	replayOrder, createdReplay, err := svc.Create(ctx, guestPrincipal, requestID, checkoutInput)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if createdReplay {
		t.Errorf("expected created = false on replay")
	}
	if replayOrder.ID != order.ID || replayOrder.OrderNumber != order.OrderNumber {
		t.Errorf("replay returned different order: %+v != %+v", replayOrder, order)
	}
	// Stock remains 3 (not decremented again)
	if store.products["680000000000000000000001"].Variants[0].Stock != 3 {
		t.Errorf("stock was modified on replay: %d", store.products["680000000000000000000001"].Variants[0].Stock)
	}

	// 4. Same key with different payload => ErrIdempotencyConflict
	diffPayload := checkoutInput
	diffPayload.ShippingAddress.Recipient = "Khách Hàng Khác"
	if _, _, err := svc.Create(ctx, guestPrincipal, requestID, diffPayload); !errors.Is(err, ErrIdempotencyConflict) {
		t.Errorf("expected ErrIdempotencyConflict, got %v", err)
	}

	// 5. Same key in a different scope runs independently
	// Reset time and price for new customer attempt
	nowTime = time.Unix(1700000000, 0)
	pCopy.Price = p1Price
	store.products["680000000000000000000001"] = pCopy

	custQuote, err := svc.Quote(ctx, checkoutInput)
	if err != nil {
		t.Fatal(err)
	}
	checkoutInput.QuoteToken = custQuote.QuoteToken

	customerPrincipal := Principal{CustomerID: "680000000000000000000002"}
	custOrder, createdCust, err := svc.Create(ctx, customerPrincipal, requestID, checkoutInput)
	if err != nil {
		t.Fatalf("Customer create with same key failed: %v", err)
	}
	if !createdCust || custOrder.ID == order.ID {
		t.Errorf("expected separate order in customer scope: %+v", custOrder)
	}

	// 6. Price change after quote => ErrQuoteChanged and stock remains unchanged
	qFresh, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qFresh.QuoteToken
	pCopy.Price = newPrice
	store.products["680000000000000000000001"] = pCopy
	stockBefore := store.products["680000000000000000000001"].Variants[0].Stock

	reqID2 := "ad577a65-cd42-46eb-ae8a-98b5f1186572"
	if _, _, err := svc.Create(ctx, guestPrincipal, reqID2, checkoutInput); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged when price changes, got %v", err)
	}
	if store.products["680000000000000000000001"].Variants[0].Stock != stockBefore {
		t.Errorf("stock changed on price change error")
	}

	// 7. Quote expired for new attempt => ErrQuoteExpired
	pCopy.Price = p1Price
	store.products["680000000000000000000001"] = pCopy
	nowTime = nowTime.Add(2 * time.Hour)
	reqID3 := "ad577a65-cd42-46eb-ae8a-98b5f1186573"
	if _, _, err := svc.Create(ctx, guestPrincipal, reqID3, checkoutInput); !errors.Is(err, ErrQuoteExpired) {
		t.Errorf("expected ErrQuoteExpired, got %v", err)
	}

	// 8. Tampered token => InvalidFieldError quote_token
	nowTime = time.Unix(1700000000, 0)
	checkoutInput.QuoteToken = checkoutInput.QuoteToken[:len(checkoutInput.QuoteToken)-2] + "xx"
	reqID4 := "ad577a65-cd42-46eb-ae8a-98b5f1186574"
	_, _, err = svc.Create(ctx, guestPrincipal, reqID4, checkoutInput)
	var ifErr *domain.InvalidFieldError
	if !errors.As(err, &ifErr) || ifErr.Field != "quote_token" {
		t.Errorf("expected InvalidFieldError on quote_token for tampered token, got %v", err)
	}

	// 9. Items mismatch token => ErrQuoteChanged
	qMatching, _ := svc.Quote(ctx, checkoutInput)
	mismatchedInput := checkoutInput
	mismatchedInput.QuoteToken = qMatching.QuoteToken
	mismatchedInput.Items[0].Quantity = 1 // token had 2
	reqID5 := "ad577a65-cd42-46eb-ae8a-98b5f1186575"
	if _, _, err := svc.Create(ctx, guestPrincipal, reqID5, mismatchedInput); !errors.Is(err, ErrQuoteChanged) {
		t.Errorf("expected ErrQuoteChanged for mismatched items, got %v", err)
	}

	// 10. Out of stock on create => ErrOutOfStock and no order created
	checkoutInput.Items[0].Quantity = 2
	qValid, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qValid.QuoteToken
	// Set stock to 0
	pCopy.Stock = 0
	pCopy.Variants[0].Stock = 0
	store.products["680000000000000000000001"] = pCopy
	reqID6 := "ad577a65-cd42-46eb-ae8a-98b5f1186576"
	if _, _, err := svc.Create(ctx, guestPrincipal, reqID6, checkoutInput); !errors.Is(err, ErrOutOfStock) {
		t.Errorf("expected ErrOutOfStock, got %v", err)
	}

	// 11. Duplicate attempt on race (ErrDuplicateAttempt) loads the committed order
	pCopy.Stock = 10
	pCopy.Variants[0].Stock = 10
	store.products["680000000000000000000001"] = pCopy
	qRace, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qRace.QuoteToken
	reqID7 := "ad577a65-cd42-46eb-ae8a-98b5f1186577"

	firstRun := true
	store.insertOrderHook = func(ctx context.Context, o domain.Order) (domain.Order, error) {
		if firstRun {
			firstRun = false
			// Simulate concurrent attempt finishing first and committing to DB
			o.ID = "690000000000000000000099"
			store.findAttemptHook = func(ctx context.Context, scope, requestID string) (*domain.Order, error) {
				return &o, nil
			}
			return domain.Order{}, ErrDuplicateAttempt
		}
		return o, nil
	}
	raceOrder, createdRace, err := svc.Create(ctx, guestPrincipal, reqID7, checkoutInput)
	store.insertOrderHook = nil
	store.findAttemptHook = nil
	if err != nil {
		t.Fatalf("expected ErrDuplicateAttempt to be handled, got %v", err)
	}
	if createdRace || raceOrder.ID != "690000000000000000000099" {
		t.Errorf("expected order from concurrent attempt to be returned: %+v", raceOrder)
	}

	// 12. Duplicate order number retries up to 3 times
	dupCount := 0
	store.insertOrderHook = func(ctx context.Context, o domain.Order) (domain.Order, error) {
		if dupCount < 2 {
			dupCount++
			return domain.Order{}, ErrDuplicateOrderNumber
		}
		return o, nil
	}
	reqID8 := "ad577a65-cd42-46eb-ae8a-98b5f1186578"
	qRetry, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qRetry.QuoteToken
	orderNumBefore := orderNumCounter
	_, _, err = svc.Create(ctx, guestPrincipal, reqID8, checkoutInput)
	store.insertOrderHook = nil
	if err != nil {
		t.Fatalf("expected retry on duplicate order number to succeed: %v", err)
	}
	if orderNumCounter-orderNumBefore != 3 {
		t.Errorf("expected 3 order numbers generated, got %d", orderNumCounter-orderNumBefore)
	}

	// 13. Unknown commit error + FindByAttempt finds order => returns order
	reqID9 := "ad577a65-cd42-46eb-ae8a-98b5f1186579"
	qUnknown, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qUnknown.QuoteToken
	store.runCheckoutHook = func(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error {
		// Run fn successfully to commit to store, but simulate connection error on return
		snapshot := store.cloneProducts()
		tx := &fakeTx{store: store, stockSnapshot: snapshot}
		_ = fn(ctx, tx)
		return errors.New("connection reset by peer")
	}
	unknownOrder, createdUnknown, err := svc.Create(ctx, guestPrincipal, reqID9, checkoutInput)
	store.runCheckoutHook = nil
	if err != nil {
		t.Fatalf("expected order to be recovered after unknown commit error: %v", err)
	}
	if createdUnknown || unknownOrder.OrderNumber == "" {
		t.Errorf("expected recovered order: %+v", unknownOrder)
	}

	// 14. Unknown commit error + FindByAttempt does NOT find order => ErrOrderUnavailable
	reqID10 := "ad577a65-cd42-46eb-ae8a-98b5f118657a"
	qFail, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qFail.QuoteToken
	store.runCheckoutHook = func(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error {
		return errors.New("disk failure")
	}
	if _, _, err := svc.Create(ctx, guestPrincipal, reqID10, checkoutInput); !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected ErrOrderUnavailable, got %v", err)
	}
	store.runCheckoutHook = nil

	// 15. hook RunCheckout calls fn TWICE consecutively (first call fails and is rolled back)
	// => exactly one order created, stock decremented only once
	pCopy.Stock = 10
	pCopy.Variants[0].Stock = 10
	store.products["680000000000000000000001"] = pCopy
	qTwice, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qTwice.QuoteToken
	reqIDTwice := "ad577a65-cd42-46eb-ae8a-98b5f1186580"

	fnCalls := 0
	store.runCheckoutHook = func(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error {
		// Call 1: fails
		fnCalls++
		snapProd1 := store.cloneProducts()
		snapOrders1, snapAtt1 := store.cloneOrders()
		tx1 := &fakeTx{store: store, stockSnapshot: snapProd1}
		_ = fn(ctx, tx1)
		// Rollback first attempt
		store.restoreProducts(snapProd1)
		store.restoreOrders(snapOrders1, snapAtt1)

		// Call 2: succeeds
		fnCalls++
		snapProd2 := store.cloneProducts()
		snapOrders2, snapAtt2 := store.cloneOrders()
		tx2 := &fakeTx{store: store, stockSnapshot: snapProd2}
		err := fn(ctx, tx2)
		if err != nil {
			store.restoreProducts(snapProd2)
			store.restoreOrders(snapOrders2, snapAtt2)
			return err
		}
		return nil
	}

	orderTwice, createdTwice, err := svc.Create(ctx, guestPrincipal, reqIDTwice, checkoutInput)
	store.runCheckoutHook = nil
	if err != nil {
		t.Fatalf("expected create with retried fn to succeed: %v", err)
	}
	if !createdTwice || orderTwice.OrderNumber == "" {
		t.Errorf("expected created = true, got %+v", orderTwice)
	}
	if fnCalls != 2 {
		t.Errorf("expected fn to be called twice, got %d", fnCalls)
	}
	// Stock started at 10, quantity was 2, should be 8 (NOT 6!)
	if store.products["680000000000000000000001"].Variants[0].Stock != 8 {
		t.Errorf("stock should be 8, got %d", store.products["680000000000000000000001"].Variants[0].Stock)
	}

	// 16. FindByAttempt fails at start of Create => ErrOrderUnavailable and does not call RunCheckout
	reqIDFindErr := "ad577a65-cd42-46eb-ae8a-98b5f1186581"
	qFindErr, _ := svc.Quote(ctx, checkoutInput)
	checkoutInput.QuoteToken = qFindErr.QuoteToken

	runCheckoutCalled := false
	store.runCheckoutHook = func(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error {
		runCheckoutCalled = true
		return nil
	}
	store.findAttemptHook = func(ctx context.Context, scope, requestID string) (*domain.Order, error) {
		return nil, errors.New("database connection refused")
	}

	_, _, err = svc.Create(ctx, guestPrincipal, reqIDFindErr, checkoutInput)
	store.findAttemptHook = nil
	store.runCheckoutHook = nil
	if !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected ErrOrderUnavailable when FindByAttempt fails at start of Create, got %v", err)
	}
	if runCheckoutCalled {
		t.Errorf("RunCheckout should NOT be called when FindByAttempt fails")
	}
}

func TestGetListAttemptVerifiedPurchase(t *testing.T) {
	price, _ := domain.ParseMoney("100000")
	products := map[string]Product{
		"680000000000000000000001": {
			ID:    "680000000000000000000001",
			Name:  "Áo dài",
			SKU:   "AO-DAI",
			Price: price,
			Stock: 10,
		},
	}
	store := newFakeStore(products)
	cat := &fakeCatalog{products: products}
	svc := NewService(cat, store, testSigner(t), 15*time.Minute, nil, nil, nil)
	ctx := context.Background()

	// Seed customer order and guest order
	custOrder := domain.Order{
		ID:          "690000000000000000000001",
		OrderNumber: "AD-CUST-01",
		CustomerID:  "680000000000000000000002",
		Status:      domain.StatusDelivered,
		Items: []domain.OrderItem{
			{ProductID: "680000000000000000000001", SKU: "AO-DAI", Quantity: 1, UnitPrice: price},
		},
		Checkout: domain.OrderCheckout{
			Scope:       "customer:680000000000000000000002",
			RequestID:   "ad577a65-cd42-46eb-ae8a-98b5f1186571",
			RequestHash: "hash1",
		},
	}
	store.ordersByID[custOrder.ID] = custOrder
	store.ordersByAttempt[custOrder.Checkout.Scope+"|"+custOrder.Checkout.RequestID] = custOrder

	guestOrder := domain.Order{
		ID:              "690000000000000000000002",
		OrderNumber:     "AD-GUEST-01",
		GuestAccessHash: "630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd",
		Status:          domain.StatusPending,
		Items: []domain.OrderItem{
			{ProductID: "680000000000000000000001", SKU: "AO-DAI", Quantity: 1, UnitPrice: price},
		},
		Checkout: domain.OrderCheckout{
			Scope:       "guest:630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd",
			RequestID:   "ad577a65-cd42-46eb-ae8a-98b5f1186572",
			RequestHash: "hash2",
		},
	}
	store.ordersByID[guestOrder.ID] = guestOrder
	store.ordersByAttempt[guestOrder.Checkout.Scope+"|"+guestOrder.Checkout.RequestID] = guestOrder

	// 1. Get tests
	// Owner customer can read
	o, err := svc.Get(ctx, Viewer{CustomerID: "680000000000000000000002"}, custOrder.ID)
	if err != nil || o == nil {
		t.Fatalf("owner customer should read order: %v", err)
	}

	// Other customer gets NotFound
	if _, err := svc.Get(ctx, Viewer{CustomerID: "680000000000000000000099"}, custOrder.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for different customer, got %v", err)
	}

	// Admin can read any order
	adminViewer := Viewer{IsAdmin: true}
	if o, err := svc.Get(ctx, adminViewer, custOrder.ID); err != nil || o == nil {
		t.Errorf("admin should read customer order: %v", err)
	}
	if o, err := svc.Get(ctx, adminViewer, guestOrder.ID); err != nil || o == nil {
		t.Errorf("admin should read guest order: %v", err)
	}

	// Guest with matching hash can read guest order
	guestViewer := Viewer{GuestHash: "630dcd2966c4336691125448bbb25b4ff412a49c732db2c8abc1b8581bd710dd"}
	if o, err := svc.Get(ctx, guestViewer, guestOrder.ID); err != nil || o == nil {
		t.Errorf("matching guest should read guest order: %v", err)
	}

	// Guest with wrong hash gets NotFound
	wrongGuest := Viewer{GuestHash: "wronghash"}
	if _, err := svc.Get(ctx, wrongGuest, guestOrder.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for wrong guest hash, got %v", err)
	}

	// Guest hash cannot read customer order
	if _, err := svc.Get(ctx, guestViewer, custOrder.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for guest attempting to read customer order, got %v", err)
	}

	// 2. List clamping
	orders, total, page, pageSize, err := svc.List(ctx, "680000000000000000000002", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page != 1 || pageSize != 20 || total != 1 || len(orders) != 1 {
		t.Errorf("unexpected list defaults: page=%d, pageSize=%d, total=%d", page, pageSize, total)
	}

	// Status filter: the customer's only order is delivered.
	if orders, total, _, _, err := svc.List(ctx, "680000000000000000000002", domain.StatusDelivered, 1, 20); err != nil || total != 1 || len(orders) != 1 {
		t.Errorf("delivered filter: total=%d len=%d err=%v", total, len(orders), err)
	}
	if orders, total, _, _, err := svc.List(ctx, "680000000000000000000002", domain.StatusPending, 1, 20); err != nil || total != 0 || len(orders) != 0 {
		t.Errorf("pending filter: total=%d len=%d err=%v", total, len(orders), err)
	}
	var invalidStatus *domain.InvalidFieldError
	if _, _, _, _, err := svc.List(ctx, "680000000000000000000002", "lost", 1, 20); !errors.As(err, &invalidStatus) || invalidStatus.Field != "status" {
		t.Errorf("unknown status error = %v", err)
	}

	_, _, _, pageSizeClamp, err := svc.List(ctx, "680000000000000000000002", "", 1, 150)
	if err != nil {
		t.Fatal(err)
	}
	if pageSizeClamp != 100 {
		t.Errorf("expected pageSize clamped to 100, got %d", pageSizeClamp)
	}

	// 3. Attempt
	// Customer attempt
	attCust, err := svc.Attempt(ctx, Viewer{CustomerID: "680000000000000000000002"}, custOrder.Checkout.RequestID)
	if err != nil || attCust == nil {
		t.Errorf("expected attempt found: %v", err)
	}
	// Wrong customer attempt => NotFound
	if _, err := svc.Attempt(ctx, Viewer{CustomerID: "680000000000000000000099"}, custOrder.Checkout.RequestID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for wrong customer attempt, got %v", err)
	}

	// Guest attempt
	attGuest, err := svc.Attempt(ctx, guestViewer, guestOrder.Checkout.RequestID)
	if err != nil || attGuest == nil {
		t.Errorf("expected guest attempt found: %v", err)
	}
	// Wrong guest attempt => NotFound
	if _, err := svc.Attempt(ctx, wrongGuest, guestOrder.Checkout.RequestID); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for wrong guest attempt, got %v", err)
	}

	// 4. VerifiedPurchase
	// Owner, delivered, product matches => true
	ok, err := svc.VerifiedPurchase(ctx, Viewer{CustomerID: "680000000000000000000002"}, custOrder.ID, "680000000000000000000001")
	if err != nil || !ok {
		t.Errorf("expected VerifiedPurchase true, got ok=%v, err=%v", ok, err)
	}

	// Wrong product => false
	ok, _ = svc.VerifiedPurchase(ctx, Viewer{CustomerID: "680000000000000000000002"}, custOrder.ID, "680000000000000000000099")
	if ok {
		t.Errorf("expected false for wrong product")
	}

	// Not delivered (guestOrder status is pending) => false
	ok, _ = svc.VerifiedPurchase(ctx, guestViewer, guestOrder.ID, "680000000000000000000001")
	if ok {
		t.Errorf("expected false for pending order")
	}

	// Admin is NOT automatically owner => false
	ok, _ = svc.VerifiedPurchase(ctx, adminViewer, custOrder.ID, "680000000000000000000001")
	if ok {
		t.Errorf("expected false for admin who is not owner")
	}

	// 5. Store error handling in Get, VerifiedPurchase, and Attempt
	// Store error on Get returns ErrOrderUnavailable
	store.findByIDHook = func(ctx context.Context, id string) (*domain.Order, error) {
		return nil, errors.New("mongodb timeout")
	}
	if _, err := svc.Get(ctx, adminViewer, custOrder.ID); !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected ErrOrderUnavailable on Get when store fails, got %v", err)
	}

	// Store error on VerifiedPurchase returns (false, ErrOrderUnavailable)
	if ok, err := svc.VerifiedPurchase(ctx, adminViewer, custOrder.ID, "680000000000000000000001"); ok || !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected (false, ErrOrderUnavailable) on VerifiedPurchase when store fails, got ok=%v, err=%v", ok, err)
	}
	store.findByIDHook = nil

	// Store error on Attempt returns ErrOrderUnavailable
	store.findAttemptHook = func(ctx context.Context, scope, requestID string) (*domain.Order, error) {
		return nil, errors.New("mongodb timeout")
	}
	if _, err := svc.Attempt(ctx, guestViewer, guestOrder.Checkout.RequestID); !errors.Is(err, ErrOrderUnavailable) {
		t.Errorf("expected ErrOrderUnavailable on Attempt when store fails, got %v", err)
	}
	store.findAttemptHook = nil

	// 6. Attempt requestID validation & normalization
	// Uppercase UUID in Attempt successfully finds order
	upperReqID := strings.ToUpper(guestOrder.Checkout.RequestID)
	attUpper, err := svc.Attempt(ctx, guestViewer, upperReqID)
	if err != nil || attUpper == nil || attUpper.ID != guestOrder.ID {
		t.Errorf("expected Attempt with uppercase UUID to find order, got %v", err)
	}

	// Malformed UUID in Attempt returns *InvalidFieldError with field 'request_id'
	_, err = svc.Attempt(ctx, guestViewer, "malformed-not-uuid")
	var ifErrAtt *domain.InvalidFieldError
	if !errors.As(err, &ifErrAtt) || ifErrAtt.Field != "request_id" {
		t.Errorf("expected InvalidFieldError on field 'request_id' for malformed UUID in Attempt, got %v", err)
	}
}
