// Removes ONLY records with the deterministic PD-01 demo slugs.
// Does not remove original application/migration data.
const database = db.getSiblingDB('QuanLyBanQuanAo');
const products = database.products.deleteMany({slug: /^pd01-ao-dai-\d{2}$/});
const categories = database.categories.deleteMany({slug: /^pd01-(co-phuc|dan-gian|cach-tan|hy-su|hidden-category)$/});
print(`Removed PD-01 demo products: ${products.deletedCount}; categories: ${categories.deletedCount}`);
