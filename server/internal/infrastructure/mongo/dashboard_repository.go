package mongo

import (
	"context"
	"fmt"

	app "github.com/your-org/aodora/server/internal/application/dashboard"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

// DashboardRepository runs the statistics aggregation pipelines.
// Money may be stored as Decimal128 (seed/checkout) or int/long (importer), so every
// monetary expression goes through $toDecimal and is rounded back with $toLong($round).
type DashboardRepository struct{ orders, customers, products, tailoring *driver.Collection }

func NewDashboardRepository(db *driver.Database) *DashboardRepository {
	return &DashboardRepository{orders: db.Collection("orders"), customers: db.Collection("customers"), products: db.Collection("products"), tailoring: db.Collection("tailoring_requests")}
}

// isRevenue: paid and not cancelled.
var isRevenue = bson.D{{Key: "$and", Value: bson.A{
	bson.D{{Key: "$eq", Value: bson.A{"$payment_status", "paid"}}},
	bson.D{{Key: "$ne", Value: bson.A{"$status", "cancelled"}}},
}}}

func roundLong(expr any) bson.D {
	return bson.D{{Key: "$toLong", Value: bson.D{{Key: "$round", Value: bson.A{expr, 0}}}}}
}
func cond(c, a, b any) bson.D {
	return bson.D{{Key: "$cond", Value: bson.A{c, a, b}}}
}
func revenueOf(field string) bson.D {
	return cond(isRevenue, bson.D{{Key: "$toDecimal", Value: field}}, 0)
}
func matchRange(field string, from, to any) bson.D {
	return bson.D{{Key: "$match", Value: bson.D{{Key: field, Value: bson.D{{Key: "$gte", Value: from}, {Key: "$lt", Value: to}}}}}}
}
func countByKey(key string) bson.A {
	return bson.A{
		bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: key}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
	}
}

type keyCount struct {
	ID      any   `bson:"_id"`
	Count   int64 `bson:"count"`
	Revenue int64 `bson:"revenue"`
}

func toCounts(in []keyCount) []app.Count {
	out := make([]app.Count, 0, len(in))
	for _, c := range in {
		out = append(out, app.Count{Key: fmt.Sprint(valueOrEmpty(c.ID)), Count: c.Count, Revenue: c.Revenue})
	}
	return out
}
func valueOrEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

func (r *DashboardRepository) Aggregate(ctx context.Context, w app.Window) (app.Raw, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	raw := app.Raw{}
	if err := r.orderStats(ctx, w, &raw); err != nil {
		return raw, err
	}
	if err := r.itemStats(ctx, w, &raw); err != nil {
		return raw, err
	}
	if err := r.inventory(ctx, w, &raw); err != nil {
		return raw, err
	}
	var err error
	if raw.Current.NewCustomers, err = r.customers.CountDocuments(ctx, bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: w.Start}, {Key: "$lt", Value: w.End}}}}); err != nil {
		return raw, err
	}
	if raw.Previous.NewCustomers, err = r.customers.CountDocuments(ctx, bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: w.PrevStart}, {Key: "$lt", Value: w.PrevEnd}}}}); err != nil {
		return raw, err
	}
	cur, err := r.tailoring.Aggregate(ctx, bson.A{matchRange("created_at", w.Start, w.End), bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$status"}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}}, bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}}})
	if err != nil {
		return raw, err
	}
	var tailoring []keyCount
	if err := cur.All(ctx, &tailoring); err != nil {
		return raw, err
	}
	raw.Tailoring = toCounts(tailoring)
	return raw, nil
}

func (r *DashboardRepository) orderStats(ctx context.Context, w app.Window, raw *app.Raw) error {
	truncate := bson.D{{Key: "date", Value: "$created_at"}, {Key: "unit", Value: string(w.Granularity)}, {Key: "timezone", Value: "+07:00"}}
	if w.Granularity == app.Week {
		truncate = append(truncate, bson.E{Key: "startOfWeek", Value: "monday"})
	}
	current := bson.D{{Key: "$match", Value: bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: w.Start}}}}}}
	pipeline := bson.A{
		matchRange("created_at", w.PrevStart, w.End),
		bson.D{{Key: "$facet", Value: bson.D{
			{Key: "kpi", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: cond(bson.D{{Key: "$gte", Value: bson.A{"$created_at", w.Start}}}, "cur", "prev")},
					{Key: "orders", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "cancelled", Value: bson.D{{Key: "$sum", Value: cond(bson.D{{Key: "$eq", Value: bson.A{"$status", "cancelled"}}}, 1, 0)}}},
					{Key: "paid", Value: bson.D{{Key: "$sum", Value: cond(isRevenue, 1, 0)}}},
					{Key: "revenue", Value: bson.D{{Key: "$sum", Value: revenueOf("$total")}}},
				}}},
				bson.D{{Key: "$project", Value: bson.D{{Key: "orders", Value: 1}, {Key: "cancelled", Value: 1}, {Key: "paid", Value: 1}, {Key: "revenue", Value: roundLong("$revenue")}}}},
			}},
			{Key: "series", Value: bson.A{
				current,
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: bson.D{{Key: "$dateTrunc", Value: truncate}}},
					{Key: "orders", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "revenue", Value: bson.D{{Key: "$sum", Value: revenueOf("$total")}}},
				}}},
				bson.D{{Key: "$project", Value: bson.D{{Key: "orders", Value: 1}, {Key: "revenue", Value: roundLong("$revenue")}}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
			}},
			{Key: "by_status", Value: append(bson.A{current}, countByKey("$status")...)},
			{Key: "by_payment_status", Value: append(bson.A{current}, countByKey("$payment_status")...)},
			{Key: "by_method", Value: bson.A{
				current,
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: "$payment_method"},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "revenue", Value: bson.D{{Key: "$sum", Value: revenueOf("$total")}}},
				}}},
				bson.D{{Key: "$project", Value: bson.D{{Key: "count", Value: 1}, {Key: "revenue", Value: roundLong("$revenue")}}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
			}},
		}}},
	}
	cur, err := r.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	var docs []struct {
		KPI []struct {
			ID        string `bson:"_id"`
			Orders    int64  `bson:"orders"`
			Cancelled int64  `bson:"cancelled"`
			Paid      int64  `bson:"paid"`
			Revenue   int64  `bson:"revenue"`
		} `bson:"kpi"`
		Series []struct {
			ID      bson.DateTime `bson:"_id"`
			Orders  int64         `bson:"orders"`
			Revenue int64         `bson:"revenue"`
		} `bson:"series"`
		ByStatus []keyCount `bson:"by_status"`
		ByPay    []keyCount `bson:"by_payment_status"`
		ByMethod []keyCount `bson:"by_method"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return err
	}
	if len(docs) == 0 {
		return nil
	}
	d := docs[0]
	for _, k := range d.KPI {
		p := app.Period{Orders: k.Orders, Cancelled: k.Cancelled, PaidOrders: k.Paid, Revenue: k.Revenue}
		if k.ID == "cur" {
			raw.Current = p
		} else {
			raw.Previous = p
		}
	}
	for _, s := range d.Series {
		raw.Series = append(raw.Series, app.Bucket{Start: s.ID.Time(), Orders: s.Orders, Revenue: s.Revenue})
	}
	raw.ByStatus, raw.ByPaymentStatus, raw.ByPaymentMethod = toCounts(d.ByStatus), toCounts(d.ByPay), toCounts(d.ByMethod)
	return nil
}

func (r *DashboardRepository) itemStats(ctx context.Context, w app.Window, raw *app.Raw) error {
	lookupProduct := func(extra bson.A) bson.D {
		return bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "products"}, {Key: "localField", Value: "_id"}, {Key: "foreignField", Value: "_id"}, {Key: "as", Value: "p"},
			{Key: "pipeline", Value: append(bson.A{bson.D{{Key: "$project", Value: bson.D{
				{Key: "name", Value: 1}, {Key: "slug", Value: 1}, {Key: "category_id", Value: 1},
				{Key: "image", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$image.src", bson.D{{Key: "$arrayElemAt", Value: bson.A{"$images", 0}}}}}}},
			}}}}, extra...)},
		}}}
	}
	byProduct := func(sortField string) bson.A {
		return bson.A{
			bson.D{{Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$items.product_id"}, {Key: "name", Value: bson.D{{Key: "$first", Value: "$items.product_name"}}},
				{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$line"}}}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}},
			}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: -1}, {Key: "_id", Value: 1}}}},
			bson.D{{Key: "$limit", Value: 10}},
			lookupProduct(nil),
			bson.D{{Key: "$project", Value: bson.D{
				{Key: "name", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$p.name", 0}}}, "$name"}}}},
				{Key: "slug", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$p.slug", 0}}}},
				{Key: "image", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$p.image", 0}}}},
				{Key: "revenue", Value: roundLong("$revenue")}, {Key: "quantity", Value: 1},
			}}},
			bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: -1}, {Key: "_id", Value: 1}}}},
		}
	}
	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: w.Start}, {Key: "$lt", Value: w.End}}},
			{Key: "payment_status", Value: "paid"}, {Key: "status", Value: bson.D{{Key: "$ne", Value: "cancelled"}}},
		}}},
		bson.D{{Key: "$unwind", Value: "$items"}},
		bson.D{{Key: "$addFields", Value: bson.D{{Key: "line", Value: bson.D{{Key: "$multiply", Value: bson.A{bson.D{{Key: "$toDecimal", Value: "$items.unit_price"}}, bson.D{{Key: "$toLong", Value: "$items.quantity"}}}}}}}}},
		bson.D{{Key: "$facet", Value: bson.D{
			{Key: "top_revenue", Value: byProduct("revenue")},
			{Key: "top_quantity", Value: byProduct("quantity")},
			{Key: "by_category", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: "$items.product_id"}, {Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$line"}}}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}}}}},
				lookupProduct(nil),
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$p.category_id", 0}}}},
					{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$revenue"}}}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$quantity"}}},
				}}},
				bson.D{{Key: "$lookup", Value: bson.D{{Key: "from", Value: "categories"}, {Key: "localField", Value: "_id"}, {Key: "foreignField", Value: "_id"}, {Key: "as", Value: "c"}}}},
				bson.D{{Key: "$project", Value: bson.D{
					{Key: "name", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$c.name", 0}}}, "Chưa phân loại"}}}},
					{Key: "revenue", Value: roundLong("$revenue")}, {Key: "quantity", Value: 1},
				}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "revenue", Value: -1}, {Key: "name", Value: 1}}}},
			}},
			{Key: "by_size", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$items.size", ""}}}},
					{Key: "revenue", Value: bson.D{{Key: "$sum", Value: "$line"}}}, {Key: "quantity", Value: bson.D{{Key: "$sum", Value: "$items.quantity"}}},
				}}},
				bson.D{{Key: "$project", Value: bson.D{{Key: "revenue", Value: roundLong("$revenue")}, {Key: "quantity", Value: 1}}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "revenue", Value: -1}, {Key: "_id", Value: 1}}}},
			}},
		}}},
	}
	cur, err := r.orders.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	type product struct {
		ID       any    `bson:"_id"`
		Name     string `bson:"name"`
		Slug     string `bson:"slug"`
		Image    string `bson:"image"`
		Revenue  int64  `bson:"revenue"`
		Quantity int64  `bson:"quantity"`
	}
	var docs []struct {
		TopRevenue  []product `bson:"top_revenue"`
		TopQuantity []product `bson:"top_quantity"`
		ByCategory  []struct {
			ID       any    `bson:"_id"`
			Name     string `bson:"name"`
			Revenue  int64  `bson:"revenue"`
			Quantity int64  `bson:"quantity"`
		} `bson:"by_category"`
		BySize []struct {
			ID       string `bson:"_id"`
			Revenue  int64  `bson:"revenue"`
			Quantity int64  `bson:"quantity"`
		} `bson:"by_size"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return err
	}
	if len(docs) == 0 {
		return nil
	}
	convert := func(in []product) []app.ProductStat {
		out := make([]app.ProductStat, 0, len(in))
		for _, p := range in {
			out = append(out, app.ProductStat{ProductID: idString(p.ID), Name: p.Name, Slug: p.Slug, Image: p.Image, Revenue: p.Revenue, Quantity: p.Quantity})
		}
		return out
	}
	d := docs[0]
	raw.TopRevenue, raw.TopQuantity = convert(d.TopRevenue), convert(d.TopQuantity)
	for _, c := range d.ByCategory {
		raw.ByCategory = append(raw.ByCategory, app.CategoryStat{CategoryID: idString(c.ID), Name: c.Name, Revenue: c.Revenue, Quantity: c.Quantity})
	}
	for _, s := range d.BySize {
		raw.BySize = append(raw.BySize, app.SizeStat{Size: s.ID, Revenue: s.Revenue, Quantity: s.Quantity})
	}
	return nil
}

func idString(v any) string {
	switch id := v.(type) {
	case bson.ObjectID:
		return id.Hex()
	case nil:
		return ""
	default:
		return fmt.Sprint(id)
	}
}

func (r *DashboardRepository) inventory(ctx context.Context, w app.Window, raw *app.Raw) error {
	raw.Inventory.LowStockThreshold = w.LowStock
	cur, err := r.products.Aggregate(ctx, bson.A{bson.D{{Key: "$facet", Value: bson.D{
		{Key: "counts", Value: bson.A{bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "$ne", Value: bson.A{"$is_active", false}}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}}}},
		{Key: "value", Value: bson.A{
			bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "value", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$multiply", Value: bson.A{bson.D{{Key: "$toDecimal", Value: "$price"}}, bson.D{{Key: "$toLong", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$stock", 0}}}}}}}}}}}}}},
			bson.D{{Key: "$project", Value: bson.D{{Key: "value", Value: roundLong("$value")}}}},
		}},
	}}}})
	if err != nil {
		return err
	}
	var inv []struct {
		Counts []struct {
			ID    bool  `bson:"_id"`
			Count int64 `bson:"count"`
		} `bson:"counts"`
		Value []struct {
			Value int64 `bson:"value"`
		} `bson:"value"`
	}
	if err := cur.All(ctx, &inv); err != nil {
		return err
	}
	if len(inv) > 0 {
		for _, c := range inv[0].Counts {
			if c.ID {
				raw.Inventory.ActiveProducts = c.Count
			} else {
				raw.Inventory.HiddenProducts = c.Count
			}
		}
		if len(inv[0].Value) > 0 {
			raw.Inventory.StockValue = inv[0].Value[0].Value
		}
	}
	// Low stock list/counts are for products that are on sale.
	cur, err = r.products.Aggregate(ctx, bson.A{
		bson.D{{Key: "$match", Value: bson.D{{Key: "is_active", Value: bson.D{{Key: "$ne", Value: false}}}}}},
		bson.D{{Key: "$unwind", Value: "$variants"}},
		bson.D{{Key: "$match", Value: bson.D{{Key: "variants.stock", Value: bson.D{{Key: "$lte", Value: w.LowStock}}}}}},
		bson.D{{Key: "$facet", Value: bson.D{
			{Key: "counts", Value: bson.A{bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "$lte", Value: bson.A{"$variants.stock", 0}}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}}}},
			{Key: "list", Value: bson.A{
				bson.D{{Key: "$sort", Value: bson.D{{Key: "variants.stock", Value: 1}, {Key: "name", Value: 1}, {Key: "variants.sku", Value: 1}}}},
				bson.D{{Key: "$limit", Value: 20}},
				bson.D{{Key: "$project", Value: bson.D{{Key: "name", Value: 1}, {Key: "slug", Value: 1}, {Key: "variants", Value: 1}}}},
			}},
		}}},
	})
	if err != nil {
		return err
	}
	var low []struct {
		Counts []struct {
			ID    bool  `bson:"_id"`
			Count int64 `bson:"count"`
		} `bson:"counts"`
		List []struct {
			ID       any    `bson:"_id"`
			Name     string `bson:"name"`
			Slug     string `bson:"slug"`
			Variants struct {
				SKU   string `bson:"sku"`
				Color string `bson:"color"`
				Size  string `bson:"size"`
				Stock int64  `bson:"stock"`
			} `bson:"variants"`
		} `bson:"list"`
	}
	if err := cur.All(ctx, &low); err != nil {
		return err
	}
	if len(low) > 0 {
		for _, c := range low[0].Counts {
			if c.ID {
				raw.Inventory.OutOfStockSKUs = c.Count
			}
			raw.Inventory.LowStockSKUs += c.Count
		}
		for _, item := range low[0].List {
			raw.Inventory.LowStock = append(raw.Inventory.LowStock, app.LowStockItem{
				ProductID: idString(item.ID), Name: item.Name, Slug: item.Slug,
				SKU: item.Variants.SKU, Color: item.Variants.Color, Size: item.Variants.Size, Stock: item.Variants.Stock,
			})
		}
	}
	return nil
}
