const start = ISODate("2026-11-20T17:00:00Z"), end = ISODate("2026-11-28T17:00:00Z");
const prevStart = ISODate("2026-11-12T17:00:00Z"), prevEnd = start;
const isRevenue = { $and: [ { $eq: ["$payment_status", "paid"] }, { $ne: ["$status", "cancelled"] } ] };
const revenueOf = f => ({ $cond: [isRevenue, { $toDecimal: f }, 0] });
const r1 = db.orders.aggregate([
  { $match: { created_at: { $gte: prevStart, $lt: end } } },
  { $facet: {
    kpi: [
      { $group: { _id: { $cond: [{ $gte: ["$created_at", start] }, "cur", "prev"] },
          orders: { $sum: 1 },
          cancelled: { $sum: { $cond: [{ $eq: ["$status", "cancelled"] }, 1, 0] } },
          paid: { $sum: { $cond: [isRevenue, 1, 0] } },
          revenue: { $sum: revenueOf("$total") } } },
      { $project: { orders: 1, cancelled: 1, paid: 1, revenue: { $toLong: { $round: ["$revenue", 0] } } } }
    ],
    series: [
      { $match: { created_at: { $gte: start } } },
      { $group: { _id: { $dateTrunc: { date: "$created_at", unit: "day", timezone: "+07:00" } },
          orders: { $sum: 1 }, revenue: { $sum: revenueOf("$total") } } },
      { $project: { orders: 1, revenue: { $toLong: { $round: ["$revenue", 0] } } } },
      { $sort: { _id: 1 } }
    ],
    by_status: [ { $match: { created_at: { $gte: start } } }, { $group: { _id: "$status", count: { $sum: 1 } } }, { $sort: { count: -1, _id: 1 } } ],
    by_payment_status: [ { $match: { created_at: { $gte: start } } }, { $group: { _id: "$payment_status", count: { $sum: 1 } } }, { $sort: { count: -1, _id: 1 } } ],
    by_method: [
      { $match: { created_at: { $gte: start } } },
      { $group: { _id: "$payment_method", count: { $sum: 1 }, revenue: { $sum: revenueOf("$total") } } },
      { $project: { count: 1, revenue: { $toLong: { $round: ["$revenue", 0] } } } },
      { $sort: { count: -1, _id: 1 } }
    ]
  } }
]).toArray();
const productLookup = { $lookup: { from: "products", localField: "_id", foreignField: "_id", as: "p",
  pipeline: [ { $project: { name: 1, slug: 1, category_id: 1, image: { $ifNull: ["$image.src", { $arrayElemAt: ["$images", 0] }] } } } ] } };
const byProduct = field => [
  { $group: { _id: "$items.product_id", name: { $first: "$items.product_name" }, revenue: { $sum: "$line" }, quantity: { $sum: "$items.quantity" } } },
  { $sort: { [field]: -1, _id: 1 } }, { $limit: 10 }, productLookup,
  { $project: { name: { $ifNull: [{ $arrayElemAt: ["$p.name", 0] }, "$name"] }, slug: { $arrayElemAt: ["$p.slug", 0] },
      image: { $arrayElemAt: ["$p.image", 0] }, revenue: { $toLong: { $round: ["$revenue", 0] } }, quantity: 1 } },
  { $sort: { [field]: -1, _id: 1 } }
];
const r2 = db.orders.aggregate([
  { $match: { created_at: { $gte: start, $lt: end }, payment_status: "paid", status: { $ne: "cancelled" } } },
  { $unwind: "$items" },
  { $addFields: { line: { $multiply: [{ $toDecimal: "$items.unit_price" }, { $toLong: "$items.quantity" }] } } },
  { $facet: {
    top_revenue: byProduct("revenue"),
    top_quantity: byProduct("quantity"),
    by_category: [
      { $group: { _id: "$items.product_id", revenue: { $sum: "$line" }, quantity: { $sum: "$items.quantity" } } },
      productLookup,
      { $group: { _id: { $arrayElemAt: ["$p.category_id", 0] }, revenue: { $sum: "$revenue" }, quantity: { $sum: "$quantity" } } },
      { $lookup: { from: "categories", localField: "_id", foreignField: "_id", as: "c" } },
      { $project: { name: { $ifNull: [{ $arrayElemAt: ["$c.name", 0] }, "Chưa phân loại"] }, revenue: { $toLong: { $round: ["$revenue", 0] } }, quantity: 1 } },
      { $sort: { revenue: -1, name: 1 } }
    ],
    by_size: [
      { $group: { _id: { $ifNull: ["$items.size", ""] }, revenue: { $sum: "$line" }, quantity: { $sum: "$items.quantity" } } },
      { $project: { revenue: { $toLong: { $round: ["$revenue", 0] } }, quantity: 1 } },
      { $sort: { revenue: -1, _id: 1 } }
    ]
  } }
]).toArray();
const r3 = db.products.aggregate([ { $facet: {
  counts: [ { $group: { _id: { $ne: ["$is_active", false] }, count: { $sum: 1 } } } ],
  value: [ { $group: { _id: null, value: { $sum: { $multiply: [{ $toDecimal: "$price" }, { $toLong: { $ifNull: ["$stock", 0] } }] } } } },
           { $project: { value: { $toLong: { $round: ["$value", 0] } } } } ]
} } ]).toArray();
const lowStock = 3;
const r4 = db.products.aggregate([
  { $match: { is_active: { $ne: false } } },
  { $unwind: "$variants" },
  { $match: { "variants.stock": { $lte: lowStock } } },
  { $facet: {
    counts: [ { $group: { _id: { $lte: ["$variants.stock", 0] }, count: { $sum: 1 } } } ],
    list: [ { $sort: { "variants.stock": 1, name: 1, "variants.sku": 1 } }, { $limit: 20 }, { $project: { name: 1, slug: 1, variants: 1 } } ]
  } }
]).toArray();
const newCustomers = db.customers.countDocuments({ created_at: { $gte: start, $lt: end } });
const tailoring = db.tailoring_requests.aggregate([ { $match: { created_at: { $gte: start, $lt: end } } }, { $group: { _id: "$status", count: { $sum: 1 } } }, { $sort: { count: -1, _id: 1 } } ]).toArray();
print(JSON.stringify({kpi:r1[0].kpi, days:r1[0].series.length, status:r1[0].by_status, top:r2[0].top_revenue.slice(0,2).map(x=>x.name), cat:r2[0].by_category.length, size:r2[0].by_size, inv:r3, low:r4[0].counts, newCustomers, tailoring}));
