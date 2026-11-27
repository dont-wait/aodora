//go:build integration

package mongo

import (
	"context"
	"strings"
	"testing"

	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAdminImportCommitAgainstMongo(t *testing.T) {
	db := scratchDatabase(t, "aodora_import_test_")
	ctx := context.Background()
	svc := app.NewService(NewAdminCatalogRepository(db))
	parse := func(text string, allowed, required []string) []app.Row {
		rows, errs := app.ParseRows([]byte(text), "csv", allowed, required)
		if len(errs) > 0 {
			t.Fatalf("parse: %+v", errs)
		}
		return rows
	}
	cats := parse("name,slug,description,parent_slug,sort_order,is_active,image_url,image_alt\nCon,import-test-con,,import-test-cha,2,true,,\nCha,import-test-cha,Cha,,1,true,https://x/c.jpg,Ảnh\n",
		app.CategoryColumns(), []string{"name", "slug"})
	rep, err := svc.ImportCategories(ctx, cats, true)
	if err != nil || !rep.Committed || rep.Created != 2 || len(rep.Errors) != 0 {
		t.Fatalf("categories: %+v err=%v", rep, err)
	}
	var child struct {
		ParentID bson.ObjectID `bson:"parent_id"`
	}
	var parent struct {
		ID bson.ObjectID `bson:"_id"`
	}
	if err := db.Collection("categories").FindOne(ctx, bson.M{"slug": "import-test-con"}).Decode(&child); err != nil {
		t.Fatal(err)
	}
	if err := db.Collection("categories").FindOne(ctx, bson.M{"slug": "import-test-cha"}).Decode(&parent); err != nil || child.ParentID != parent.ID {
		t.Fatalf("parent link: child=%v parent=%v err=%v", child.ParentID, parent.ID, err)
	}

	header := "product_slug,name,category_slug,description,material,price,compare_at_price,is_active,image_url,gallery_urls,color_name,color_hex,size,sku,stock\n"
	required := []string{"product_slug", "name", "category_slug", "price", "sku"}
	first := parse(header+"import-test-ao,Áo thử,import-test-cha,Mô tả,Lụa,1500000,1800000,true,https://x/a.jpg,,Đỏ,#ff0000,S,IMPORT-TEST-AO-S,5\nimport-test-ao,,,,,,,,,,Đỏ,#ff0000,M,IMPORT-TEST-AO-M,2\n", app.ProductColumns(), required)
	rep, err = svc.ImportProducts(ctx, first, true)
	if err != nil || !rep.Committed || rep.Created != 1 || len(rep.Errors) != 0 {
		t.Fatalf("products: %+v err=%v", rep, err)
	}
	stockOf := func(sku string) int64 {
		var doc struct {
			Stock    int64 `bson:"stock"`
			Version  int64 `bson:"metadata_version"`
			Variants []struct {
				SKU   string `bson:"sku"`
				Stock int64  `bson:"stock"`
			} `bson:"variants"`
		}
		if err := db.Collection("products").FindOne(ctx, bson.M{"slug": "import-test-ao"}).Decode(&doc); err != nil {
			t.Fatal(err)
		}
		for _, v := range doc.Variants {
			if v.SKU == sku {
				return v.Stock
			}
		}
		return -1
	}
	if stockOf("IMPORT-TEST-AO-S") != 5 || stockOf("IMPORT-TEST-AO-M") != 2 {
		t.Fatal("opening stock not stored")
	}
	// Second import: new price, existing SKU with a different stock (ignored), one new SKU with stock.
	second := parse(header+"import-test-ao,Áo thử,import-test-cha,Mô tả,Lụa,1600000,1800000,true,https://x/a.jpg,,Đỏ,#ff0000,S,IMPORT-TEST-AO-S,999\nimport-test-ao,,,,,,,,,,Xanh,,L,IMPORT-TEST-AO-L,4\n", app.ProductColumns(), required)
	dry, err := svc.ImportProducts(ctx, second, false)
	if err != nil || len(dry.Errors) != 0 || dry.Updated != 1 || dry.IgnoredStock != 1 || dry.Committed {
		t.Fatalf("dry run: %+v err=%v", dry, err)
	}
	rep, err = svc.ImportProducts(ctx, second, true)
	if err != nil || !rep.Committed || rep.Updated != 1 {
		t.Fatalf("second import: %+v err=%v", rep, err)
	}
	if got := stockOf("IMPORT-TEST-AO-S"); got != 5 {
		t.Fatalf("existing SKU stock was overwritten: %d", got)
	}
	if got := stockOf("IMPORT-TEST-AO-L"); got != 4 {
		t.Fatalf("new SKU opening stock = %d, want 4", got)
	}
	var doc bson.M
	if err := db.Collection("products").FindOne(ctx, bson.M{"slug": "import-test-ao"}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc["stock"] == nil || toInt(doc["stock"]) != 11 {
		t.Fatalf("aggregate stock must be 5+2+4=11, got %v", doc["stock"])
	}
	if !strings.Contains(strings.ToLower(toStr(doc["name"])), "áo thử") || toInt(doc["price"]) != 1600000 {
		t.Fatalf("price not updated: %v", doc)
	}
	// Re-running the same file is a no-op.
	again, err := svc.ImportProducts(ctx, second, true)
	if err != nil || again.Updated != 0 || again.Skipped != 1 {
		t.Fatalf("idempotent: %+v err=%v", again, err)
	}
	// Export contains the three SKUs.
	rows, err := svc.ExportProducts(ctx)
	if err != nil || len(rows) != 4 {
		t.Fatalf("export rows = %d err=%v", len(rows), err)
	}
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int32:
		return int64(n)
	case int64:
		return n
	case bson.Decimal128:
		s := n.String()
		var out int64
		for _, c := range s {
			if c < '0' || c > '9' {
				break
			}
			out = out*10 + int64(c-'0')
		}
		return out
	case float64:
		return int64(n)
	}
	return -1
}
func toStr(v any) string { s, _ := v.(string); return s }
