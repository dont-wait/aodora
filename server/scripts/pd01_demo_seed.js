// PD-01 read-only API demonstration data. Opt-in script, NOT a migration.
// Run AFTER migrations: mongosh "mongodb://aodora:aodora@localhost:27017/QuanLyBanQuanAo?authSource=admin" --file scripts/pd01_demo_seed.js
// It uses $setOnInsert and deterministic IDs; rerunning does not overwrite edits.
const database = db.getSiblingDB('QuanLyBanQuanAo');
const now = new Date();
const categories = [
  ['7f0000000000000000000001', 'co-phuc', 'pd01-co-phuc', 'Cổ Phục Ngũ Thân', 1, true],
  ['7f0000000000000000000002', 'dan-gian', 'pd01-dan-gian', 'Dân Gian & Lễ Hội', 2, true],
  ['7f0000000000000000000003', 'cach-tan', 'pd01-cach-tan', 'Cách Tân Hiện Đại', 3, true],
  ['7f0000000000000000000004', 'hy-su', 'pd01-hy-su', 'Áo Dài Cưới & Hỷ Sự', 4, true],
  ['7f0000000000000000000005', null, 'pd01-hidden-category', 'Danh mục ẩn PD01', 5, false],
];
for (const [hex, key, slug, name, order, active] of categories) {
  const entry = {_id: ObjectId(hex), name, slug, description: 'Dữ liệu minh họa PD-01',
    parent_id: null, is_active: active, sort_order: order, created_at: now, updated_at: now};
  if (key) entry.id = key;
  database.categories.updateOne({slug}, {$setOnInsert: entry}, {upsert: true});
}

const images = ["https://lh3.googleusercontent.com/aida-public/AB6AXuBKP9Blq_vA0B_9bZjlPNuaHCqYdWu_OjwERzAAUybzkqUDpGsvGbTlGO3oPIboYLVI8BN3UE1cbhc-zWsaP5slNhDC48Kyu6Go3g4T21sSbRl2kolndGu9yCtRSsx2xpwZZYq5rjdHqAuuKBTC8sKTTk5CerK55Y9tehSvDH8y2dHvHd4cIDrUXdOz2C0spWdpqQp-TL6LmHzYULkNzy4tJL61kMRZ9qxEmbuqJveRJfPlOmnOs2oRKg", "https://lh3.googleusercontent.com/aida-public/AB6AXuAddueeXxpyJDroCuHZy0cru4FG0izQnLO34h2CdWe_t40aTK9KK5sesqD_RWmK1boDTXzXe1Et9VRRu37y7c2_V5yEqTFmKJQS_5_J_W84utdNrasWgIPLoI57DE3NEgLM4MQfFrTETSKI4GlKTKk6XU1e8AQ2THvm3IaP77LN50OAaStn_YFw952gHTgNUMxedGoKKd75v_c9eqbdwOKIYYxSW08oQGFb5prTaqLDSEJJeCEnDPjwaA", "https://lh3.googleusercontent.com/aida-public/AB6AXuAbZqRv8xlhlTFNsCUaHTklJ9zahvMU0CR5YnDnf0QUdApQ3jTGcvp2kPVDJZG0lVuo9j59MbSazsc33XCh0uUdGCmD292juBgbiuskazK-aTLmCQMpeLR3KhSMD2SKAIehKlmwfBcnnmukMdt47udLwFlaKejFe6ExdBoTInRZcEdMuPC1WpfHzeD5AkBTJ8NivMF-5fQ4pFQuQ9a1ytNEjPPVcKLRezp8WLF4y-TC63idhmD5SIn2LQ", "https://lh3.googleusercontent.com/aida-public/AB6AXuAOs1LNDvBxKr-dO-sO4CY52wokcMGkjcm9_8YKBVZ-OKjy-HizijouDbsvFivYGhnFrBELSI0nLb8-hJ1BWAQdPkYGFBsv23TKSwQm8kzq5svCVRZUFbazpWzGaQNJqcK0ipXZIDZyFdd7jtJDL9zrumxjdkXCb94mDS1l-G7E1jznCk-rQ2e2qTgKL5ae7VHFdl-VPMfcVo41SGS2LM2tIHJDE9zZ3wiG177Xb_YgrVWWKGBVRlS0fg", "https://lh3.googleusercontent.com/aida-public/AB6AXuCivf5J3wjfVKpsQLM3XikmfFXoXed5GcKuG1gxPpRDe36C5v-7N-QNcDLQCLoUZDSQnHRZZWBimgDbW0CRgIdmVuE30P3Dvv6M3aZfGSMwTj-ZYntDmfEESI8qkopgWAEROzDyKEEDG3To4OXvl4ZEVMALTbdUc473PrtcVVhRmEY89zREFCGPWqI6tVV1S-LkII3tpIku8Oz4Zk9aMkRWcHUu5QwdBs51OGDooltsDAQx1o9NXfY8bg", "https://lh3.googleusercontent.com/aida-public/AB6AXuAE4K2XggWeXHB-ZztkdvFVgTUML8wpp2FJyY4m2fqjHdjPGWDSSx9UyG1PLpj0_NbX1iK25-alPWRbF7QIT9EntkamDrKl5sPzou_AlFBGVmexfJO_duoPDQJH83QHvtp1JnT5FEiYO07xo4zDAAADRZNUcMbsiv9cRi400NOOHxdI2NI92ksFp_tYBXQa3kJzpWMyY8qKePW3Lp7gA8De3UoH6KnFDuYKq4blRBgdEfjo9_rayFDeiw", "https://lh3.googleusercontent.com/aida-public/AB6AXuAtq7qPqT5lNAelS1tqNNDoqggKSGrsjQnjEfBYBNXlQkLXtJuD7nglj9fqJHymRaw95OPHe-9lbeBKpJ4xrnbfiRgPGkKh3etcMVoTnyq8eEIdI9DjAK9hWDZ-AJxVtYLVe5qdQJZmW7BwjUVRLrO3MslzhQ8_MSuwcAaQzz_T_SUp97YsaeIgjcaZswOI2w_0ZWYyTJCNq3sCMWMEzOFtiHBrRm9Kt9WcxzYd7ZeAPETm31jHzXbCcA", "https://lh3.googleusercontent.com/aida-public/AB6AXuDKKOO-4XZITrPgy7BUaDQavdM9w4pAlM-2-i5ReIkOEDQAjcG9NfoGwajIrDyPI_xSxlLwFmqP5NA7cnQxwVTTL_grFK6R7Fo7Fs4HFw37-ys7AZIqZ3A0Qzq5CXwjzn4zNqsuOGXdYvLL0m2YacWZmNq4Z-uWy9KEy9-U0otjImL6DuR1rtNvAtNEg1uY4BNHdIDd91pNTAk39EevhISef9wdIRRHbhRF2UJrcpvnbh1pOhxjoDUrTg"];
const catalog = [
  ['Áo Dài Lụa Ngà Lập Lĩnh', 0, 2950000, 'Lụa Tơ Tằm', true],
  ['Áo Dài Gấm Trầm Hoa Sen', 0, 3250000, 'Gấm Dệt', true],
  ['Áo Dài Ngũ Thân Mộc', 0, 2150000, 'Vải Đũi', true],
  ['Áo Dài Trắng Hoa Cúc', 0, 2750000, 'Lụa Sa Hàn', true],
  ['Áo Dài Mai Điểu', 1, 3200000, 'Lụa Tơ Tằm', true],
  ['Áo Dài Dân Gian Cánh Sen', 1, 2450000, 'Gấm Dệt', true],
  ['Áo Dài Cách Tân Xanh Ngọc', 2, 1950000, 'Lụa Đũi', false],
  ['Áo Dài Cách Tân Tay Lỡ', 2, 1750000, 'Lụa Sa Hàn', false],
  ['Áo Dài Cách Tân Vàng Mơ', 2, 2190000, 'Lụa Đũi', false],
  ['Áo Dài Cách Tân Hồng Phấn', 2, 2250000, 'Lụa Sa Hàn', false],
  ['Áo Dài Hỷ Sự Phượng Hoàng', 3, 5500000, 'Gấm Dệt', true],
  ['Áo Dài Cưới Chỉ Kim', 3, 5200000, 'Lụa Tơ Tằm', true],
  ['Áo Dài Đỏ Tơ Tằm', 1, 3500000, 'Lụa Tơ Tằm', true],
  ['Áo Dài Phom Xưa Dệt Nổi', 0, 3850000, 'Gấm Dệt', true],
  ['Áo Dài Khăn Vấn Đồng Sắc', 1, 2650000, 'Vải Đũi', true],
  ['Áo Dài Ngọc Bích Lụa Mộc', 2, 1980000, 'Lụa Đũi', false],
  ['Áo Dài Tạm Ngưng Bán', 0, 1990000, 'Lụa Tơ Tằm', false],
  ['Áo Dài Thuộc Danh Mục Ẩn', 4, 1990000, 'Lụa Tơ Tằm', false],
];

for (let i = 0; i < catalog.length; i++) {
  const [name, category, price, material, bespoke] = catalog[i];
  const ordinal = String(i + 1).padStart(2, '0');
  const slug = `pd01-ao-dai-${ordinal}`;
  // Intentional SKU mismatch: Ngà / M and Đỏ / L exist; Ngà / L does not.
  // Product 02 has no ready-to-buy stock; product 17 is inactive.
  const variants = [
    {sku: `PD01-${ordinal}-NGA-M`, color: 'Ngà', size: 'M', stock: i === 1 ? 0 : 3},
    {sku: `PD01-${ordinal}-DO-L`, color: 'Đỏ', size: 'L', stock: i === 1 ? 0 : 2},
  ];
  const totalStock = variants.reduce((sum, item) => sum + item.stock, 0);
  const image = {src: images[i % images.length], alt: name};
  const document = {
    _id: ObjectId(`7e00000000000000000000${ordinal}`), slug, name,
    category_id: ObjectId(categories[category][0]),
    description: `Mẫu áo dài ${name.toLowerCase()} cho thử nghiệm tìm kiếm PD-01`,
    price: NumberLong(String(price)), stock: totalStock, variants,
    material, image, images: [image.src], gallery: [image],
    colors: [{name:'Ngà',hex:'#FAF5EE'}, {name:'Đỏ',hex:'#7A1C29'}],
    sizes: ['M','L'], badge: {label: 'Mẫu minh họa',variant: 'surface'},
    bespoke, is_active: i !== 16,
    story: {eyebrow:'Cảm hứng thiết kế', title:`Câu chuyện của ${name}`,
      paragraphs: ['Mẫu thử nghiệm PD-01 để kiểm tra nội dung theo slug.', 'Mỗi áo có nội dung riêng được lưu trong MongoDB.'], image},
    fabric: {title:'Chất liệu', description: material, care: [{title:'Giặt nhẹ',description:'Giặt tay với nước mát.'}]},
    craft: {title:'Quy trình thực hiện', steps:[{title:'Đo và cắt',description:'Chuẩn bị theo kích cỡ chọn.'}]},
    accessories: [], trust: [],
    created_at: new Date(now.getTime() - i * 60000), updated_at: now,
  };
  database.products.updateOne({slug}, {$setOnInsert: document}, {upsert: true});
}
print(`PD-01 seed complete: ${catalog.length} demo products (16 visible, 1 inactive, 1 in a hidden category) and ${categories.length} categories. Existing records were not overwritten.`);
