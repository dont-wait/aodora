//go:build integration

package mongo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	catalogadmin "github.com/your-org/aodora/server/internal/application/catalogadmin"
	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/infrastructure/migration"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	sharedTestClientMu sync.Mutex
	sharedTestClient   *mongo.Client
)

func getSharedTestClient(t *testing.T, rawURI string) *mongo.Client {
	t.Helper()
	sharedTestClientMu.Lock()
	defer sharedTestClientMu.Unlock()
	if sharedTestClient != nil {
		return sharedTestClient
	}
	parsedURI, err := url.Parse(rawURI)
	if err != nil {
		t.Fatalf("parse MONGO_TEST_URI: %v", err)
	}
	opts := options.Client().ApplyURI(parsedURI.String()).SetMaxPoolSize(20)
	client, err := mongo.Connect(opts)
	if err != nil {
		t.Fatalf("connect shared test client: %v", err)
	}
	sharedTestClient = client
	return sharedTestClient
}

func orderTestDatabase(t *testing.T) (*mongo.Client, *mongo.Database, string) {
	t.Helper()
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI must point to the isolated replica set; integration tests must not silently skip")
	}
	parsedURI, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse MONGO_TEST_URI: %v", err)
	}

	name := "aodora_dw02_test_" + bson.NewObjectID().Hex()
	parsedURI.Path = "/" + name
	targetURI := parsedURI.String()

	client := getSharedTestClient(t, raw)
	db := client.Database(name)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := db.Drop(ctx); err != nil {
			t.Errorf("drop isolated test database %s: %v", name, err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var hello bson.M
	if err := db.RunCommand(ctx, bson.D{{Key: "hello", Value: 1}}).Decode(&hello); err != nil {
		t.Fatalf("hello command: %v", err)
	}
	if hello["setName"] == nil || hello["isWritablePrimary"] != true {
		t.Fatalf("tests require a writable replica set: %v", hello)
	}

	migrationsPath := os.Getenv("MONGO_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "../../../migrations"
	}
	if err := migration.Run(targetURI, os.DirFS(migrationsPath), "up"); err != nil {
		t.Fatalf("run migrations on test db: %v", err)
	}

	// Clear any demo seed data so tests start with a completely clean state
	for _, col := range []string{"orders", "products", "categories", "customers"} {
		if _, err := db.Collection(col).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatalf("clear demo seed %s: %v", col, err)
		}
	}

	return client, db, targetURI
}

func seedCategory(t *testing.T, db *mongo.Database, name, slug string, parentID *bson.ObjectID, isActive bool) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	uniqueSlug := fmt.Sprintf("%s-%s", slug, id.Hex()[:8])
	now := time.Now().UTC().Truncate(time.Millisecond)
	doc := bson.M{
		"_id":        id,
		"name":       name,
		"slug":       uniqueSlug,
		"is_active":  isActive,
		"created_at": now,
		"updated_at": now,
	}
	if parentID != nil {
		doc["parent_id"] = *parentID
	} else {
		doc["parent_id"] = nil
	}
	_, err := db.Collection("categories").InsertOne(context.Background(), doc)
	if err != nil {
		t.Fatalf("seedCategory failed: %v", err)
	}
	return id
}

func seedProduct(t *testing.T, db *mongo.Database, catID bson.ObjectID, name, slug string, price any, stock int64, variants bson.A, isActive bool) bson.ObjectID {
	t.Helper()
	id := bson.NewObjectID()
	uniqueSlug := fmt.Sprintf("%s-%s", slug, id.Hex()[:8])
	now := time.Now().UTC().Truncate(time.Millisecond)
	doc := bson.M{
		"_id":         id,
		"name":        name,
		"slug":        uniqueSlug,
		"category_id": catID,
		"price":       price,
		"stock":       stock,
		"is_active":   isActive,
		"version":     int64(1),
		"created_at":  now,
		"updated_at":  now,
	}
	if variants != nil {
		doc["variants"] = variants
	}
	_, err := db.Collection("products").InsertOne(context.Background(), doc)
	if err != nil {
		t.Fatalf("seedProduct failed: %v", err)
	}
	return id
}

func newTestService(t *testing.T, client *mongo.Client, db *mongo.Database, orderRepo *OrderRepository, feeStr string, nowFunc func() time.Time, newOrderNum func() string) *apporder.Service {
	t.Helper()
	signer, err := domainorder.NewQuoteSigner([]byte("01234567890123456789012345678901"))
	if err != nil {
		t.Fatalf("new quote signer: %v", err)
	}
	var codFee *domainorder.Money
	if feeStr != "" {
		parsed, err := domainorder.ParseMoney(feeStr)
		if err != nil {
			t.Fatalf("parse shipping fee %s: %v", feeStr, err)
		}
		codFee = &parsed
	}
	return apporder.NewService(
		orderRepo,
		orderRepo,
		signer,
		15*time.Minute,
		codFee,
		nowFunc,
		newOrderNum,
	)
}

func clearOrders(t *testing.T, db *mongo.Database) {
	t.Helper()
	if _, err := db.Collection("orders").DeleteMany(context.Background(), bson.M{}); err != nil {
		t.Fatalf("clear orders: %v", err)
	}
}

// Suite 1: Checkout Scenarios (cases 1-9)
func TestOrderCheckoutScenarios(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	repo := NewOrderRepository(client, db)

	t.Run("GuestCashAtCounterOnlyNamePhone", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Ao Nam", "ao-nam", nil, true)
		prodID := seedProduct(t, db, catID, "Ao So Mi", "ao-so-mi", int64(250000), 10, bson.A{
			bson.M{"sku": "SM-L", "size": "L", "color": "Trang", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)

		input := domainorder.CheckoutInput{
			PaymentMethod: "cash_at_counter",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Van Guest",
				Phone:     "0912345678",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "SM-L", Quantity: 1},
			},
		}

		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote failed: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		guestHash := "1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff"
		principal := apporder.Principal{GuestHash: guestHash}
		reqID := "b1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

		order, created, err := svc.Create(ctx, principal, reqID, input)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if !created {
			t.Fatalf("expected created = true")
		}
		if order.Status != "pending" || order.PaymentStatus != "unpaid" {
			t.Fatalf("expected pending/unpaid, got status=%s, payment_status=%s", order.Status, order.PaymentStatus)
		}
		if order.ShippingFee.String() != "0" {
			t.Fatalf("expected shipping_fee 0 for cash_at_counter, got %s", order.ShippingFee.String())
		}

		orderOID, err := oid(order.ID)
		if err != nil {
			t.Fatalf("invalid order ID: %v", err)
		}
		var rawDoc bson.M
		if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": orderOID}).Decode(&rawDoc); err != nil {
			t.Fatalf("find raw order: %v", err)
		}

		if _, exists := rawDoc["customer_id"]; exists {
			t.Fatalf("customer_id field MUST be absent for guest, got: %v", rawDoc["customer_id"])
		}
		if asString(rawDoc["guest_access_hash"]) != guestHash {
			t.Fatalf("expected guest_access_hash %s, got %v", guestHash, rawDoc["guest_access_hash"])
		}
		checkoutMap := asMap(rawDoc["checkout"])
		if checkoutMap == nil {
			t.Fatalf("checkout field must exist")
		}
		if asString(checkoutMap["scope"]) != "guest:"+guestHash {
			t.Fatalf("expected scope guest:%s, got %v", guestHash, checkoutMap["scope"])
		}

		custCount, err := db.Collection("customers").CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatalf("count customers: %v", err)
		}
		if custCount != 0 {
			t.Fatalf("expected 0 customers in database, got %d", custCount)
		}
	})

	t.Run("CustomerCODDecimalAndIntegerPrice", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Vay Nu", "vay-nu", nil, true)

		decPrice, err := bson.ParseDecimal128("199999.5")
		if err != nil {
			t.Fatalf("parse decimal128: %v", err)
		}
		prodDecID := seedProduct(t, db, catID, "Vay Hoa Decimal", "vay-hoa-decimal", decPrice, 10, bson.A{
			bson.M{"sku": "VH-M", "size": "M", "color": "Hong", "stock": int64(10)},
		}, true)

		prodIntID := seedProduct(t, db, catID, "Vay Hoa Int", "vay-hoa-int", int64(200000), 10, bson.A{
			bson.M{"sku": "VI-M", "size": "M", "color": "Do", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		inputDec := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Tran Thi B",
				Phone:     "0987654321",
				Line1:     "456 Duong 2",
				City:      "Ha Noi",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodDecID.Hex(), SKU: "VH-M", Quantity: 3},
			},
		}
		quoteDec, err := svc.Quote(ctx, inputDec)
		if err != nil {
			t.Fatalf("Quote decimal failed: %v", err)
		}
		if quoteDec.Subtotal.String() != "599998.5" {
			t.Fatalf("expected subtotal 599998.5, got %s", quoteDec.Subtotal.String())
		}
		inputDec.QuoteToken = quoteDec.QuoteToken

		orderDec, _, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, "c1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", inputDec)
		if err != nil {
			t.Fatalf("Create decimal failed: %v", err)
		}

		orderDecOID, _ := oid(orderDec.ID)
		var rawDecDoc bson.M
		if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": orderDecOID}).Decode(&rawDecDoc); err != nil {
			t.Fatalf("find raw decimal order: %v", err)
		}

		if _, ok := rawDecDoc["subtotal"].(bson.Decimal128); !ok {
			t.Fatalf("subtotal should be stored as Decimal128, got %T: %v", rawDecDoc["subtotal"], rawDecDoc["subtotal"])
		}
		items := asSlice(rawDecDoc["items"])
		item0 := asMap(items[0])
		if _, ok := item0["unit_price"].(bson.Decimal128); !ok {
			t.Fatalf("unit_price should be stored as Decimal128, got %T: %v", item0["unit_price"], item0["unit_price"])
		}

		readDec, err := repo.FindByID(ctx, orderDec.ID)
		if err != nil {
			t.Fatalf("FindByID decimal failed: %v", err)
		}
		if readDec.Subtotal.String() != "599998.5" || readDec.Items[0].UnitPrice.String() != "199999.5" {
			t.Fatalf("read back mismatch: subtotal=%s, unit_price=%s", readDec.Subtotal.String(), readDec.Items[0].UnitPrice.String())
		}

		inputInt := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Tran Thi B",
				Phone:     "0987654321",
				Line1:     "456 Duong 2",
				City:      "Ha Noi",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodIntID.Hex(), SKU: "VI-M", Quantity: 2},
			},
		}
		quoteInt, err := svc.Quote(ctx, inputInt)
		if err != nil {
			t.Fatalf("Quote int failed: %v", err)
		}
		inputInt.QuoteToken = quoteInt.QuoteToken

		orderInt, _, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, "d1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", inputInt)
		if err != nil {
			t.Fatalf("Create int failed: %v", err)
		}

		orderIntOID, _ := oid(orderInt.ID)
		var rawIntDoc bson.M
		if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": orderIntOID}).Decode(&rawIntDoc); err != nil {
			t.Fatalf("find raw int order: %v", err)
		}
		if _, ok := rawIntDoc["subtotal"].(int64); !ok {
			t.Fatalf("integer price order subtotal should be stored as int64, got %T: %v", rawIntDoc["subtotal"], rawIntDoc["subtotal"])
		}
	})

	t.Run("SequentialReplayWithSameKeyAndPayload", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Quan Nam", "quan-nam", nil, true)
		prodID := seedProduct(t, db, catID, "Quan Jean", "quan-jean", int64(300000), 10, bson.A{
			bson.M{"sku": "QJ-30", "size": "30", "color": "Xanh", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()
		reqID := "e1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Le Van C",
				Phone:     "0911223344",
				Line1:     "789 Duong 3",
				City:      "Da Nang",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "QJ-30", Quantity: 2},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote failed: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		order1, created1, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if err != nil {
			t.Fatalf("First Create failed: %v", err)
		}
		if !created1 {
			t.Fatalf("expected created1 = true")
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 8 {
			t.Fatalf("expected stock 8, got %d", asInt64(prodDoc["stock"]))
		}

		_, err = db.Collection("products").UpdateOne(ctx, bson.M{"_id": prodID}, bson.M{"$set": bson.M{"price": int64(500000)}})
		if err != nil {
			t.Fatalf("update price: %v", err)
		}

		order2, created2, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if err != nil {
			t.Fatalf("Replay Create failed: %v", err)
		}
		if created2 {
			t.Fatalf("expected created2 = false on replay")
		}
		if order2.ID != order1.ID {
			t.Fatalf("expected order2.ID == order1.ID (%s != %s)", order2.ID, order1.ID)
		}

		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 8 {
			t.Fatalf("expected stock to remain 8 after replay, got %d", asInt64(prodDoc["stock"]))
		}
	})

	t.Run("ConcurrentIdempotentRequests", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Ao Khoac", "ao-khoac", nil, true)
		prodID := seedProduct(t, db, catID, "Ao Gio", "ao-gio", int64(400000), 20, bson.A{
			bson.M{"sku": "AG-XL", "size": "XL", "color": "Den", "stock": int64(20)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()
		reqID := "f1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Pham Van D",
				Phone:     "0922334455",
				Line1:     "101 Duong 4",
				City:      "Can Tho",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "AG-XL", Quantity: 2},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote failed: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		const concurrency = 8
		var wg sync.WaitGroup
		errCh := make(chan error, concurrency)
		orderIDCh := make(chan string, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				order, _, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
				if err != nil {
					errCh <- err
					return
				}
				orderIDCh <- order.ID
			}()
		}

		wg.Wait()
		close(errCh)
		close(orderIDCh)

		for err := range errCh {
			t.Fatalf("concurrent request failed: %v", err)
		}

		var firstID string
		countIDs := 0
		for id := range orderIDCh {
			countIDs++
			if firstID == "" {
				firstID = id
			} else if id != firstID {
				t.Fatalf("expected all goroutines to get same ID %s, got %s", firstID, id)
			}
		}
		if countIDs != concurrency {
			t.Fatalf("expected %d successful responses, got %d", concurrency, countIDs)
		}

		docCount, err := db.Collection("orders").CountDocuments(ctx, bson.M{"checkout.request_id": reqID})
		if err != nil {
			t.Fatalf("count orders: %v", err)
		}
		if docCount != 1 {
			t.Fatalf("expected 1 document in orders collection, got %d", docCount)
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 18 {
			t.Fatalf("expected stock 18, got %d", asInt64(prodDoc["stock"]))
		}
	})

	t.Run("IdempotencyConflictAndDifferentScopes", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Phu Kien", "phu-kien", nil, true)
		prodID := seedProduct(t, db, catID, "Mu Len", "mu-len", int64(100000), 50, bson.A{
			bson.M{"sku": "ML-01", "size": "F", "color": "Nau", "stock": int64(50)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custA := bson.NewObjectID().Hex()
		custB := bson.NewObjectID().Hex()
		reqID := "11b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

		input1 := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "User A",
				Phone:     "0933445566",
				Line1:     "111 Duong 5",
				City:      "Hue",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "ML-01", Quantity: 1},
			},
		}
		quote1, _ := svc.Quote(ctx, input1)
		input1.QuoteToken = quote1.QuoteToken

		orderA, createdA, err := svc.Create(ctx, apporder.Principal{CustomerID: custA}, reqID, input1)
		if err != nil || !createdA {
			t.Fatalf("Create orderA failed: %v", err)
		}

		inputDiff := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "User A",
				Phone:     "0933445566",
				Line1:     "111 Duong 5",
				City:      "Hue",
			},
			Note: "Different note",
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "ML-01", Quantity: 1},
			},
		}
		quoteDiff, _ := svc.Quote(ctx, inputDiff)
		inputDiff.QuoteToken = quoteDiff.QuoteToken

		_, _, err = svc.Create(ctx, apporder.Principal{CustomerID: custA}, reqID, inputDiff)
		if !errors.Is(err, apporder.ErrIdempotencyConflict) {
			t.Fatalf("expected ErrIdempotencyConflict, got: %v", err)
		}

		inputB := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "User B",
				Phone:     "0944556677",
				Line1:     "222 Duong 6",
				City:      "Vung Tau",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "ML-01", Quantity: 1},
			},
		}
		quoteB, _ := svc.Quote(ctx, inputB)
		inputB.QuoteToken = quoteB.QuoteToken

		orderB, createdB, err := svc.Create(ctx, apporder.Principal{CustomerID: custB}, reqID, inputB)
		if err != nil || !createdB {
			t.Fatalf("Create orderB in scope custB failed: %v", err)
		}
		if orderB.ID == orderA.ID {
			t.Fatalf("different scopes must create independent orders")
		}
	})

	t.Run("OutOfStockRace", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Limited Edition", "limited-edition", nil, true)
		svc := newTestService(t, client, db, repo, "17000", nil, nil)

		const rounds = 20
		for r := 0; r < rounds; r++ {
			sku := fmt.Sprintf("LIM-%d", r)
			prodID := seedProduct(t, db, catID, "Ao Hiem", fmt.Sprintf("ao-hiem-%d", r), int64(100000), 1, bson.A{
				bson.M{"sku": sku, "size": "M", "color": "Vang", "stock": int64(1)},
			}, true)

			inputA := domainorder.CheckoutInput{
				PaymentMethod: "cash_at_counter",
				ShippingAddress: domainorder.ShippingAddressInput{
					Recipient: "Guest A",
					Phone:     "0911111111",
				},
				Items: []domainorder.OrderItemInput{
					{ProductID: prodID.Hex(), SKU: sku, Quantity: 1},
				},
			}
			quoteA, err := svc.Quote(ctx, inputA)
			if err != nil {
				t.Fatalf("round %d: Quote A: %v", r, err)
			}
			inputA.QuoteToken = quoteA.QuoteToken

			inputB := domainorder.CheckoutInput{
				PaymentMethod: "cash_at_counter",
				ShippingAddress: domainorder.ShippingAddressInput{
					Recipient: "Guest B",
					Phone:     "0922222222",
				},
				Items: []domainorder.OrderItemInput{
					{ProductID: prodID.Hex(), SKU: sku, Quantity: 1},
				},
			}
			quoteB, err := svc.Quote(ctx, inputB)
			if err != nil {
				t.Fatalf("round %d: Quote B: %v", r, err)
			}
			inputB.QuoteToken = quoteB.QuoteToken

			guestHashA := fmt.Sprintf("aaaa%060d", r)
			guestHashB := fmt.Sprintf("bbbb%060d", r)
			reqIDA := fmt.Sprintf("21b2c3d4-e5f6-4a1b-8c2d-%012d", r*2)
			reqIDB := fmt.Sprintf("21b2c3d4-e5f6-4a1b-8c2d-%012d", r*2+1)

			var wg sync.WaitGroup
			var errA, errB error
			wg.Add(2)

			go func() {
				defer wg.Done()
				_, _, errA = svc.Create(ctx, apporder.Principal{GuestHash: guestHashA}, reqIDA, inputA)
			}()
			go func() {
				defer wg.Done()
				_, _, errB = svc.Create(ctx, apporder.Principal{GuestHash: guestHashB}, reqIDB, inputB)
			}()

			wg.Wait()

			successA := (errA == nil)
			successB := (errB == nil)

			if successA == successB {
				t.Fatalf("round %d: expected exactly one success, got A=%v, B=%v", r, errA, errB)
			}
			if !successA && !errors.Is(errA, apporder.ErrOutOfStock) {
				t.Fatalf("round %d: expected ErrOutOfStock for A, got: %v", r, errA)
			}
			if !successB && !errors.Is(errB, apporder.ErrOutOfStock) {
				t.Fatalf("round %d: expected ErrOutOfStock for B, got: %v", r, errB)
			}

			var prodDoc bson.M
			_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
			stock := asInt64(prodDoc["stock"])
			if stock != 0 {
				t.Fatalf("round %d: expected aggregate stock 0, got %d", r, stock)
			}
			variants := asSlice(prodDoc["variants"])
			varStock := asInt64(asMap(variants[0])["stock"])
			if varStock != 0 {
				t.Fatalf("round %d: expected variant stock 0, got %d", r, varStock)
			}
		}
	})

	t.Run("OrderThreeProductsLastOOS", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Bo San Pham", "bo-san-pham", nil, true)
		p1 := seedProduct(t, db, catID, "SP 1", "sp-1", int64(100000), 5, bson.A{
			bson.M{"sku": "SKU-1", "size": "M", "color": "Trang", "stock": int64(5)},
		}, true)
		p2 := seedProduct(t, db, catID, "SP 2", "sp-2", int64(150000), 5, bson.A{
			bson.M{"sku": "SKU-2", "size": "M", "color": "Den", "stock": int64(5)},
		}, true)
		p3 := seedProduct(t, db, catID, "SP 3", "sp-3", int64(200000), 1, bson.A{
			bson.M{"sku": "SKU-3", "size": "M", "color": "Do", "stock": int64(1)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Khach Hang",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: p1.Hex(), SKU: "SKU-1", Quantity: 2},
				{ProductID: p2.Hex(), SKU: "SKU-2", Quantity: 2},
				{ProductID: p3.Hex(), SKU: "SKU-3", Quantity: 1},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		_, err = db.Collection("products").UpdateOne(ctx, bson.M{"_id": p3}, bson.M{
			"$set": bson.M{"stock": 0, "variants.0.stock": 0},
		})
		if err != nil {
			t.Fatalf("reduce p3 stock: %v", err)
		}

		reqID := "31b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"
		_, _, err = svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if !errors.Is(err, apporder.ErrOutOfStock) {
			t.Fatalf("expected ErrOutOfStock, got: %v", err)
		}

		count, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"checkout.request_id": reqID})
		if count != 0 {
			t.Fatalf("expected 0 orders, got %d", count)
		}

		var doc1, doc2 bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": p1}).Decode(&doc1)
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": p2}).Decode(&doc2)
		if asInt64(doc1["stock"]) != 5 || asInt64(doc2["stock"]) != 5 {
			t.Fatalf("expected stock 5 for p1 and p2, got p1=%d, p2=%d", asInt64(doc1["stock"]), asInt64(doc2["stock"]))
		}
	})

	t.Run("DuplicateOrderNumberRollbackAndRetry", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Danh Muc Test", "danh-muc-test", nil, true)
		prodID := seedProduct(t, db, catID, "Ao Test", "ao-test", int64(120000), 10, bson.A{
			bson.M{"sku": "AT-01", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		existingOrderNumber := "AD-261116-COLLIS01"
		_, err := db.Collection("orders").InsertOne(ctx, bson.M{
			"_id":            bson.NewObjectID(),
			"order_number":   existingOrderNumber,
			"status":         "pending",
			"payment_method": "cod",
			"payment_status": "unpaid",
			"shipping_fee":   int64(0),
			"subtotal":       int64(100000),
			"total":          int64(100000),
			"items": bson.A{
				bson.M{
					"product_id":   prodID,
					"product_name": "Ao Pre-exist",
					"sku":          "AT-01",
					"quantity":     int64(1),
					"unit_price":   int64(100000),
				},
			},
			"shipping_address": bson.M{
				"recipient": "Pre Exist",
				"phone":     "0900000000",
				"line1":     "Pre Line",
				"city":      "Pre City",
			},
			"created_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("insert existing order: %v", err)
		}

		callCount := 0
		generator := func() string {
			callCount++
			if callCount == 1 {
				return existingOrderNumber
			}
			return "AD-261116-UNIQUE02"
		}

		svc := newTestService(t, client, db, repo, "17000", nil, generator)
		custID := bson.NewObjectID().Hex()
		reqID := "41b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Van Retry",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "AT-01", Quantity: 2},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		order, created, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if err != nil {
			t.Fatalf("Create after retry failed: %v", err)
		}
		if !created {
			t.Fatalf("expected created = true")
		}
		if order.OrderNumber != "AD-261116-UNIQUE02" {
			t.Fatalf("expected order number AD-261116-UNIQUE02, got %s", order.OrderNumber)
		}
		if callCount < 2 {
			t.Fatalf("expected at least 2 calls to order number generator, got %d", callCount)
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 8 {
			t.Fatalf("expected stock 8 (decremented once), got %d (transaction rollback failed)", asInt64(prodDoc["stock"]))
		}
	})

	t.Run("PriceOrStockChangedAfterQuote", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "DanhMuc9", "danh-muc-9", nil, true)
		prodID := seedProduct(t, db, catID, "Ao Thun 9", "ao-thun-9", int64(100000), 10, bson.A{
			bson.M{"sku": "AT-09", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguoi Mua",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "AT-09", Quantity: 3},
			},
		}
		quote, _ := svc.Quote(ctx, input)
		input.QuoteToken = quote.QuoteToken

		_, err := db.Collection("products").UpdateOne(ctx, bson.M{"_id": prodID}, bson.M{"$set": bson.M{"price": int64(120000)}})
		if err != nil {
			t.Fatalf("update price: %v", err)
		}

		_, _, err = svc.Create(ctx, apporder.Principal{CustomerID: custID}, "51b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", input)
		if !errors.Is(err, apporder.ErrQuoteChanged) {
			t.Fatalf("expected ErrQuoteChanged, got: %v", err)
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 10 {
			t.Fatalf("expected stock 10, got %d", asInt64(prodDoc["stock"]))
		}

		_, _ = db.Collection("products").UpdateOne(ctx, bson.M{"_id": prodID}, bson.M{"$set": bson.M{"price": int64(100000)}})
		input.Items[0].Quantity = 5
		quote2, _ := svc.Quote(ctx, input)
		input.QuoteToken = quote2.QuoteToken

		_, _ = db.Collection("products").UpdateOne(ctx, bson.M{"_id": prodID}, bson.M{
			"$set": bson.M{"stock": 2, "variants.0.stock": 2},
		})

		_, _, err = svc.Create(ctx, apporder.Principal{CustomerID: custID}, "52b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", input)
		if !errors.Is(err, apporder.ErrOutOfStock) {
			t.Fatalf("expected ErrOutOfStock, got: %v", err)
		}
	})
}

// Suite 2: Catalog & Admin Concurrency Scenarios (cases 10-14)
func TestOrderCatalogAndAdminScenarios(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	repo := NewOrderRepository(client, db)
	adminRepo := NewAdminCatalogRepository(db)

	t.Run("ConcurrentOrdersWithAdminInventoryAdjustmentAndUpdateProduct", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "DanhMuc10", "danh-muc-10", nil, true)
		sku := "SKU-10"
		initialStock := int64(50)
		prodID := seedProduct(t, db, catID, "Ao Khoac Concurrency", "ao-khoac-concurrency", int64(100000), initialStock, bson.A{
			bson.M{"sku": sku, "size": "L", "color": "Xanh", "stock": initialStock},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)

		input := domainorder.CheckoutInput{
			PaymentMethod: "cash_at_counter",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Guest",
				Phone:     "0912345678",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: sku, Quantity: 1},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		const orderCount = 5
		const adjustCount = 5
		const deltaPerAdjust = int64(2)

		var wg sync.WaitGroup
		var soldCount int64
		var orderMu sync.Mutex
		orderErrors := make([]error, 0)
		var adminMu sync.Mutex
		adminErrors := make([]error, 0)

		for i := 0; i < orderCount; i++ {
			wg.Add(1)
			idx := i
			go func() {
				defer wg.Done()
				reqID := fmt.Sprintf("61b2c3d4-e5f6-4a1b-8c2d-%012d", idx)
				guestHash := fmt.Sprintf("6666%060d", idx)
				_, created, err := svc.Create(ctx, apporder.Principal{GuestHash: guestHash}, reqID, input)
				orderMu.Lock()
				if err != nil {
					orderErrors = append(orderErrors, err)
				} else if created {
					soldCount++
				}
				orderMu.Unlock()
			}()
		}

		for i := 0; i < adjustCount; i++ {
			wg.Add(1)
			idx := i
			go func() {
				defer wg.Done()
				reqID := fmt.Sprintf("adjust-req-%d", idx)
				_, err := adminRepo.AdjustInventory(ctx, prodID.Hex(), sku, deltaPerAdjust, reqID)
				if err != nil {
					adminMu.Lock()
					adminErrors = append(adminErrors, err)
					adminMu.Unlock()
				}
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			var curDoc bson.M
			if err := db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&curDoc); err != nil {
				adminMu.Lock()
				adminErrors = append(adminErrors, err)
				adminMu.Unlock()
				return
			}
			curVer := asInt64(curDoc["version"])
			_, err := adminRepo.UpdateProduct(ctx, prodID.Hex(), catalogadmin.ProductInput{
				Name:            "Ao Khoac Concurrency Updated",
				Slug:            asString(curDoc["slug"]),
				CategoryID:      catID.Hex(),
				Description:     "Admin updated metadata concurrently",
				Price:           100000,
				IsActive:        true,
				ExpectedVersion: &curVer,
			})
			if err != nil {
				adminMu.Lock()
				adminErrors = append(adminErrors, err)
				adminMu.Unlock()
			}
		}()

		wg.Wait()

		if len(orderErrors) > 0 {
			t.Fatalf("order errors occurred: %v", orderErrors)
		}
		if len(adminErrors) > 0 {
			t.Fatalf("admin errors occurred: %v", adminErrors)
		}
		if soldCount != int64(orderCount) {
			t.Fatalf("expected soldCount == orderCount (%d != %d)", soldCount, orderCount)
		}

		var finalDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&finalDoc)

		if asString(finalDoc["name"]) != "Ao Khoac Concurrency Updated" {
			t.Fatalf("expected admin updated name 'Ao Khoac Concurrency Updated', got '%s'", asString(finalDoc["name"]))
		}
		if asString(finalDoc["description"]) != "Admin updated metadata concurrently" {
			t.Fatalf("expected admin updated description 'Admin updated metadata concurrently', got '%s'", asString(finalDoc["description"]))
		}

		expectedStock := initialStock + (int64(adjustCount) * deltaPerAdjust) - soldCount
		actualStock := asInt64(finalDoc["stock"])
		if actualStock != expectedStock {
			t.Fatalf("expected final stock %d, got %d (sold=%d)", expectedStock, actualStock, soldCount)
		}

		variants := asSlice(finalDoc["variants"])
		var varSum int64
		for _, v := range variants {
			varSum += asInt64(asMap(v)["stock"])
		}
		if varSum != actualStock {
			t.Fatalf("variant sum %d != aggregate stock %d", varSum, actualStock)
		}
	})

	t.Run("ProductModifiedAfterOrderRetainsSnapshot", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "DanhMuc11", "danh-muc-11", nil, true)
		prodID := seedProduct(t, db, catID, "Ao So Mi Goc", "ao-so-mi-goc", int64(200000), 10, bson.A{
			bson.M{"sku": "SM-11", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Snapshot",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "SM-11", Quantity: 1},
			},
		}
		quote, _ := svc.Quote(ctx, input)
		input.QuoteToken = quote.QuoteToken

		order, _, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, "71b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d", input)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		_, err = db.Collection("products").UpdateOne(ctx, bson.M{"_id": prodID}, bson.M{
			"$set": bson.M{"name": "Ao So Mi Da Sua", "price": int64(999000)},
		})
		if err != nil {
			t.Fatalf("update product: %v", err)
		}

		retrieved, err := svc.Get(ctx, apporder.Viewer{CustomerID: custID}, order.ID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if retrieved.Items[0].ProductName != "Ao So Mi Goc" {
			t.Fatalf("expected snapshot product name 'Ao So Mi Goc', got '%s'", retrieved.Items[0].ProductName)
		}
		if retrieved.Items[0].UnitPrice.String() != "200000" {
			t.Fatalf("expected snapshot unit price 200000, got %s", retrieved.Items[0].UnitPrice.String())
		}
	})

	t.Run("InconsistentStockRejectsOrder", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "DanhMuc12", "danh-muc-12", nil, true)
		prodID := seedProduct(t, db, catID, "Ao Stock Lech", "ao-stock-lech", int64(100000), 10, bson.A{
			bson.M{"sku": "SL-12", "size": "M", "color": "Trang", "stock": int64(5)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Lech",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "SL-12", Quantity: 1},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		reqID := "81b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"
		_, _, err = svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if !errors.Is(err, apporder.ErrOrderUnavailable) {
			t.Fatalf("expected ErrOrderUnavailable, got: %v", err)
		}

		count, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"checkout.request_id": reqID})
		if count != 0 {
			t.Fatalf("expected 0 orders, got %d", count)
		}
	})

	t.Run("InactiveProductOrCategoryOrAncestor", func(t *testing.T) {
		clearOrders(t, db)
		catRoot := seedCategory(t, db, "Cat Root", "cat-root", nil, true)
		prodInactive := seedProduct(t, db, catRoot, "Ao Inactive", "ao-inactive", int64(100000), 10, bson.A{
			bson.M{"sku": "IN-01", "size": "M", "color": "Trang", "stock": int64(10)},
		}, false)

		catInactive := seedCategory(t, db, "Cat Inactive", "cat-inactive", nil, false)
		prodInInactiveCat := seedProduct(t, db, catInactive, "Ao Cat Inactive", "ao-cat-inactive", int64(100000), 10, bson.A{
			bson.M{"sku": "IN-02", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		catChild := seedCategory(t, db, "Cat Child", "cat-child", &catInactive, true)
		prodChild := seedProduct(t, db, catChild, "Ao Ancestor Inactive", "ao-ancestor-inactive", int64(100000), 10, bson.A{
			bson.M{"sku": "IN-03", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		svc := newTestService(t, client, db, repo, "17000", nil, nil)

		testCases := []struct {
			name   string
			prodID string
			sku    string
		}{
			{"InactiveProduct", prodInactive.Hex(), "IN-01"},
			{"InactiveCategory", prodInInactiveCat.Hex(), "IN-02"},
			{"InactiveAncestor", prodChild.Hex(), "IN-03"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				input := domainorder.CheckoutInput{
					PaymentMethod: "cash_at_counter",
					ShippingAddress: domainorder.ShippingAddressInput{
						Recipient: "Nguyen Test",
						Phone:     "0912345678",
					},
					Items: []domainorder.OrderItemInput{
						{ProductID: tc.prodID, SKU: tc.sku, Quantity: 1},
					},
				}
				_, err := svc.Quote(ctx, input)
				if err == nil {
					t.Fatalf("expected InvalidField error, got nil")
				}
				var invErr *domainorder.InvalidFieldError
				if !errors.As(err, &invErr) || invErr.Field != "items[0].product_id" {
					t.Fatalf("expected InvalidFieldError on items[0].product_id, got: %v", err)
				}
			})
		}
	})

	t.Run("LegacyProductWithoutVariants", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Legacy Cat", "legacy-cat", nil, true)
		prodID := bson.NewObjectID()
		now := time.Now().UTC()
		_, err := db.Collection("products").InsertOne(ctx, bson.M{
			"_id":         prodID,
			"name":        "Legacy Ao Thun",
			"slug":        fmt.Sprintf("legacy-ao-thun-%s", prodID.Hex()[:8]),
			"category_id": catID,
			"sku":         "LEGACY-SKU",
			"price":       int64(180000),
			"stock":       int64(10),
			"variants":    bson.A{},
			"is_active":   true,
			"created_at":  now,
			"updated_at":  now,
		})
		if err != nil {
			t.Fatalf("insert legacy product: %v", err)
		}

		svc := newTestService(t, client, db, repo, "17000", nil, nil)
		custID := bson.NewObjectID().Hex()

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Legacy",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "LEGACY-SKU", Quantity: 2},
			},
		}
		quote, err := svc.Quote(ctx, input)
		if err != nil {
			t.Fatalf("Quote: %v", err)
		}
		input.QuoteToken = quote.QuoteToken

		reqID := "91b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8d"
		order, created, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if err != nil || !created {
			t.Fatalf("Create legacy order: %v", err)
		}
		if order.Items[0].SKU != "LEGACY-SKU" {
			t.Fatalf("expected SKU LEGACY-SKU, got %s", order.Items[0].SKU)
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 8 {
			t.Fatalf("expected stock 8, got %d", asInt64(prodDoc["stock"]))
		}
	})
}

// Suite 3: Queries and Recovery Scenarios (cases 15, 16)
func TestOrderQueriesAndRecoveryScenarios(t *testing.T) {
	client, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	repo := NewOrderRepository(client, db)

	t.Run("ListPaginationStableAndAuthorization", func(t *testing.T) {
		clearOrders(t, db)
		fixedTime := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
		svc := newTestService(t, client, db, repo, "17000", func() time.Time { return fixedTime }, nil)

		custA := bson.NewObjectID().Hex()
		custB := bson.NewObjectID().Hex()

		orderIDsA := make([]string, 5)
		for i := 0; i < 5; i++ {
			id := bson.NewObjectID()
			orderIDsA[i] = id.Hex()
			_, err := db.Collection("orders").InsertOne(ctx, bson.M{
				"_id":            id,
				"order_number":   fmt.Sprintf("AD-A-%d", i),
				"customer_id":    oidMust(custA),
				"status":         "pending",
				"payment_method": "cod",
				"payment_status": "unpaid",
				"shipping_fee":   int64(0),
				"subtotal":       int64(100000),
				"total":          int64(100000),
				"items": bson.A{
					bson.M{
						"product_id":   bson.NewObjectID(),
						"product_name": "Product",
						"sku":          "SKU-1",
						"quantity":     int64(1),
						"unit_price":   int64(100000),
					},
				},
				"shipping_address": bson.M{
					"recipient": "Customer A",
					"phone":     "0912345678",
					"line1":     "123 Duong",
					"city":      "HCM",
				},
				"created_at": fixedTime,
				"updated_at": fixedTime,
			})
			if err != nil {
				t.Fatalf("seed order A %d: %v", i, err)
			}
		}

		idB := bson.NewObjectID()
		_, err := db.Collection("orders").InsertOne(ctx, bson.M{
			"_id":            idB,
			"order_number":   "AD-B-1",
			"customer_id":    oidMust(custB),
			"status":         "pending",
			"payment_method": "cod",
			"payment_status": "unpaid",
			"shipping_fee":   int64(0),
			"subtotal":       int64(100000),
			"total":          int64(100000),
			"items": bson.A{
				bson.M{
					"product_id":   bson.NewObjectID(),
					"product_name": "Product B",
					"sku":          "SKU-B",
					"quantity":     int64(1),
					"unit_price":   int64(100000),
				},
			},
			"shipping_address": bson.M{
				"recipient": "Customer B",
				"phone":     "0987654321",
				"line1":     "456 Duong",
				"city":      "Ha Noi",
			},
			"created_at": fixedTime,
			"updated_at": fixedTime,
		})
		if err != nil {
			t.Fatalf("seed order B: %v", err)
		}

		p1, total1, _, _, err := svc.List(ctx, custA, 1, 2)
		if err != nil || total1 != 5 || len(p1) != 2 {
			t.Fatalf("page 1 failed: err=%v, total=%d, len=%d", err, total1, len(p1))
		}

		p2, total2, _, _, err := svc.List(ctx, custA, 2, 2)
		if err != nil || total2 != 5 || len(p2) != 2 {
			t.Fatalf("page 2 failed: err=%v, total=%d, len=%d", err, total2, len(p2))
		}

		p3, total3, _, _, err := svc.List(ctx, custA, 3, 2)
		if err != nil || total3 != 5 || len(p3) != 1 {
			t.Fatalf("page 3 failed: err=%v, total=%d, len=%d", err, total3, len(p3))
		}

		seenIDs := make(map[string]bool)
		allPages := append(append(p1, p2...), p3...)
		for _, o := range allPages {
			if seenIDs[o.ID] {
				t.Fatalf("duplicate order %s across pagination pages", o.ID)
			}
			seenIDs[o.ID] = true
		}
		if len(seenIDs) != 5 {
			t.Fatalf("expected 5 distinct orders across pages, got %d", len(seenIDs))
		}

		if seenIDs[idB.Hex()] {
			t.Fatalf("custB order leaked into custA list")
		}

		_, err = svc.Get(ctx, apporder.Viewer{CustomerID: custB}, orderIDsA[0])
		if !errors.Is(err, apporder.ErrNotFound) {
			t.Fatalf("expected ErrNotFound when CustB reads CustA order, got: %v", err)
		}

		guestHash := "eeee1111222233334444555566667777888899990000aaaabbbbccccddddeeee"
		guestOrderID := bson.NewObjectID()
		_, err = db.Collection("orders").InsertOne(ctx, bson.M{
			"_id":               guestOrderID,
			"order_number":      "AD-GUEST-1",
			"guest_access_hash": guestHash,
			"status":            "pending",
			"payment_method":    "cash_at_counter",
			"payment_status":    "unpaid",
			"shipping_fee":      int64(0),
			"subtotal":          int64(100000),
			"total":             int64(100000),
			"items": bson.A{
				bson.M{
					"product_id":   bson.NewObjectID(),
					"product_name": "Product",
					"sku":          "SKU-1",
					"quantity":     int64(1),
					"unit_price":   int64(100000),
				},
			},
			"shipping_address": bson.M{
				"recipient": "Guest",
				"phone":     "0912345678",
			},
			"created_at": fixedTime,
			"updated_at": fixedTime,
		})
		if err != nil {
			t.Fatalf("seed guest order: %v", err)
		}

		wrongHash := "ffff1111222233334444555566667777888899990000aaaabbbbccccddddeeee"
		_, err = svc.Get(ctx, apporder.Viewer{GuestHash: wrongHash}, guestOrderID.Hex())
		if !errors.Is(err, apporder.ErrNotFound) {
			t.Fatalf("expected ErrNotFound for wrong guest hash, got: %v", err)
		}
	})

	t.Run("LostResponseAfterCommit", func(t *testing.T) {
		clearOrders(t, db)
		catID := seedCategory(t, db, "Cat Lost Resp", "cat-lost-resp", nil, true)
		prodID := seedProduct(t, db, catID, "Ao Lost Resp", "ao-lost-resp", int64(150000), 10, bson.A{
			bson.M{"sku": "LR-01", "size": "M", "color": "Trang", "stock": int64(10)},
		}, true)

		wrappedStore := &lostResponseStoreWrapper{
			Store:            repo,
			shouldFailCommit: true,
		}

		signer, _ := domainorder.NewQuoteSigner([]byte("01234567890123456789012345678901"))
		fee, _ := domainorder.ParseMoney("17000")
		svc := apporder.NewService(
			repo,
			wrappedStore,
			signer,
			15*time.Minute,
			&fee,
			nil,
			nil,
		)

		custID := bson.NewObjectID().Hex()
		reqID := "a1b2c3d4-e5f6-4a1b-8c2d-3e4f5a6b7c8e"

		input := domainorder.CheckoutInput{
			PaymentMethod: "cod",
			ShippingAddress: domainorder.ShippingAddressInput{
				Recipient: "Nguyen Recovery",
				Phone:     "0912345678",
				Line1:     "123 Duong",
				City:      "HCM",
			},
			Items: []domainorder.OrderItemInput{
				{ProductID: prodID.Hex(), SKU: "LR-01", Quantity: 2},
			},
		}
		quote, _ := svc.Quote(ctx, input)
		input.QuoteToken = quote.QuoteToken

		order, created, err := svc.Create(ctx, apporder.Principal{CustomerID: custID}, reqID, input)
		if err != nil {
			t.Fatalf("Create should recover committed order after network loss, got err: %v", err)
		}
		if created {
			t.Fatalf("recovered order should have created = false")
		}
		if order.ID == "" {
			t.Fatalf("expected non-empty order ID")
		}

		var prodDoc bson.M
		_ = db.Collection("products").FindOne(ctx, bson.M{"_id": prodID}).Decode(&prodDoc)
		if asInt64(prodDoc["stock"]) != 8 {
			t.Fatalf("expected stock 8, got %d", asInt64(prodDoc["stock"]))
		}

		docCount, _ := db.Collection("orders").CountDocuments(ctx, bson.M{"checkout.request_id": reqID})
		if docCount != 1 {
			t.Fatalf("expected exactly 1 order in DB, got %d", docCount)
		}
	})
}

// Suite 4: Migration order_number_unique (case 17)
func TestMigrationOrderNumberUnique(t *testing.T) {
	_, db, targetURI := orderTestDatabase(t)
	ctx := context.Background()

	migrationsPath := os.Getenv("MONGO_MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "../../../migrations"
	}
	files := os.DirFS(migrationsPath)

	downThroughMigration(t, targetURI, files, "order_number_unique")

	pID := bson.NewObjectID()
	legacy1 := bson.M{
		"_id":            bson.NewObjectID(),
		"status":         "pending",
		"payment_method": "cod",
		"payment_status": "unpaid",
		"shipping_fee":   int64(0),
		"subtotal":       int64(100000),
		"total":          int64(100000),
		"items": bson.A{
			bson.M{
				"product_id":   pID,
				"product_name": "Legacy Item 1",
				"sku":          "LEG-1",
				"quantity":     int64(1),
				"unit_price":   int64(100000),
			},
		},
		"shipping_address": bson.M{
			"recipient": "Legacy User 1",
			"phone":     "0911111111",
			"line1":     "Line 1",
			"city":      "City 1",
		},
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}
	legacy2 := bson.M{
		"_id":            bson.NewObjectID(),
		"status":         "pending",
		"payment_method": "cod",
		"payment_status": "unpaid",
		"shipping_fee":   int64(0),
		"subtotal":       int64(200000),
		"total":          int64(200000),
		"items": bson.A{
			bson.M{
				"product_id":   pID,
				"product_name": "Legacy Item 2",
				"sku":          "LEG-2",
				"quantity":     int64(1),
				"unit_price":   int64(200000),
			},
		},
		"shipping_address": bson.M{
			"recipient": "Legacy User 2",
			"phone":     "0922222222",
			"line1":     "Line 2",
			"city":      "City 2",
		},
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}

	_, err := db.Collection("orders").InsertOne(ctx, legacy1)
	if err != nil {
		t.Fatalf("insert legacy 1: %v", err)
	}
	_, err = db.Collection("orders").InsertOne(ctx, legacy2)
	if err != nil {
		t.Fatalf("insert legacy 2: %v", err)
	}

	if err := migration.Run(targetURI, files, "up"); err != nil {
		t.Fatalf("up migration on legacy orders: %v", err)
	}

	var check1, check2 bson.M
	if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": legacy1["_id"]}).Decode(&check1); err != nil {
		t.Fatalf("find legacy 1: %v", err)
	}
	if err := db.Collection("orders").FindOne(ctx, bson.M{"_id": legacy2["_id"]}).Decode(&check2); err != nil {
		t.Fatalf("find legacy 2: %v", err)
	}
	if _, exists := check1["order_number"]; exists {
		t.Fatalf("legacy 1 should not have been rewritten with order_number")
	}

	legacy3 := bson.M{
		"_id":            bson.NewObjectID(),
		"status":         "pending",
		"payment_method": "cod",
		"payment_status": "unpaid",
		"shipping_fee":   int64(0),
		"subtotal":       int64(300000),
		"total":          int64(300000),
		"items": bson.A{
			bson.M{
				"product_id":   pID,
				"product_name": "Legacy Item 3",
				"sku":          "LEG-3",
				"quantity":     int64(1),
				"unit_price":   int64(300000),
			},
		},
		"shipping_address": bson.M{
			"recipient": "Legacy User 3",
			"phone":     "0933333333",
			"line1":     "Line 3",
			"city":      "City 3",
		},
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}
	if _, err := db.Collection("orders").InsertOne(ctx, legacy3); err != nil {
		t.Fatalf("inserting 3rd order without order_number should succeed under partial index: %v", err)
	}

	dupOrderNum := "AD-TEST-DUP-UNIQUE"
	orderWithNum1 := bson.M{
		"_id":            bson.NewObjectID(),
		"order_number":   dupOrderNum,
		"status":         "pending",
		"payment_method": "cod",
		"payment_status": "unpaid",
		"shipping_fee":   int64(0),
		"subtotal":       int64(100000),
		"total":          int64(100000),
		"items": bson.A{
			bson.M{
				"product_id":   pID,
				"product_name": "Item",
				"sku":          "SKU",
				"quantity":     int64(1),
				"unit_price":   int64(100000),
			},
		},
		"shipping_address": bson.M{
			"recipient": "User",
			"phone":     "0944444444",
			"line1":     "Line",
			"city":      "City",
		},
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}
	orderWithNum2 := bson.M{
		"_id":            bson.NewObjectID(),
		"order_number":   dupOrderNum,
		"status":         "pending",
		"payment_method": "cod",
		"payment_status": "unpaid",
		"shipping_fee":   int64(0),
		"subtotal":       int64(100000),
		"total":          int64(100000),
		"items": bson.A{
			bson.M{
				"product_id":   pID,
				"product_name": "Item",
				"sku":          "SKU",
				"quantity":     int64(1),
				"unit_price":   int64(100000),
			},
		},
		"shipping_address": bson.M{
			"recipient": "User 2",
			"phone":     "0955555555",
			"line1":     "Line",
			"city":      "City",
		},
		"created_at": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
	}

	if _, err := db.Collection("orders").InsertOne(ctx, orderWithNum1); err != nil {
		t.Fatalf("insert first order with number: %v", err)
	}

	_, err = db.Collection("orders").InsertOne(ctx, orderWithNum2)
	if err == nil {
		t.Fatalf("expected duplicate key error on inserting duplicate order_number, got nil")
	}

	downThroughMigration(t, targetURI, files, "order_number_unique")

	if _, err := db.Collection("orders").InsertOne(ctx, orderWithNum2); err != nil {
		t.Fatalf("inserting duplicate after index dropped should succeed: %v", err)
	}

	_, _ = db.Collection("orders").DeleteOne(ctx, bson.M{"_id": orderWithNum2["_id"]})
	if err := migration.Run(targetURI, files, "up"); err != nil {
		t.Fatalf("re-applying up migration after cleanup: %v", err)
	}
}

// downThroughMigration rolls back every migration applied after name, then
// name itself, so later migrations do not change which one this test reverts.
func downThroughMigration(t *testing.T, targetURI string, files fs.FS, name string) {
	t.Helper()
	all, err := migration.Load(files)
	if err != nil {
		t.Fatal(err)
	}
	for i, m := range all {
		if m.Name != name {
			continue
		}
		for range all[i:] {
			if err := migration.Run(targetURI, files, "down"); err != nil {
				t.Fatalf("down migration: %v", err)
			}
		}
		return
	}
	t.Fatalf("migration %s not found", name)
}

func oidMust(hex string) bson.ObjectID {
	id, err := oid(hex)
	if err != nil {
		panic(err)
	}
	return id
}

type lostResponseStoreWrapper struct {
	apporder.Store
	shouldFailCommit bool
}

func (w *lostResponseStoreWrapper) RunCheckout(ctx context.Context, fn func(ctx context.Context, tx apporder.CheckoutTx) error) error {
	err := w.Store.RunCheckout(ctx, fn)
	if err == nil && w.shouldFailCommit {
		return errors.New("simulated network connection reset after commit")
	}
	return err
}
