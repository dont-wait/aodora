// Command importsumire imports the Sumire Store áo dài catalog into MongoDB.
// Images are uploaded separately by the rustfs-sumire compose service (make sumire-images).
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type importOptions struct {
	catalogPath, assetsPath                                      string
	check, dryRun, update, allowRemote, purge, yes, verifyTarget bool
	limit                                                        int
	defaultStock                                                 int64
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "importsumire:", err)
		os.Exit(1)
	}
}

func run() error {
	var o importOptions
	flag.StringVar(&o.catalogPath, "catalog", "seeds/sumire/catalog.json", "catalog.json downloaded from Drive")
	flag.StringVar(&o.assetsPath, "assets", "seeds/sumire", "directory holding <handle>/NN.<ext> images")
	flag.BoolVar(&o.check, "check", false, "validate catalog and image files without database access")
	flag.BoolVar(&o.dryRun, "dry-run", false, "read the database and report what would be inserted, without writing")
	flag.BoolVar(&o.update, "update", false, "refresh source fields (name, description, images, colors) of existing products; never stock, price or version")
	flag.BoolVar(&o.allowRemote, "allow-remote", false, "allow a MONGO_URI that is not localhost")
	flag.BoolVar(&o.purge, "purge", false, "delete documents created by this importer (needs -yes)")
	flag.BoolVar(&o.yes, "yes", false, "confirm -purge")
	flag.BoolVar(&o.verifyTarget, "verify-target", false, "only verify MONGO_URI is localhost (used by make before migrating)")
	flag.IntVar(&o.limit, "limit", 0, "import only the first N products (0 = all)")
	flag.Int64Var(&o.defaultStock, "default-stock", 5, "stock given to each available variant (source has no quantities)")
	flag.Parse()
	if o.defaultStock < 0 || o.defaultStock > maxDefaultStock || o.limit < 0 {
		return fmt.Errorf("-default-stock must be 0..%d and -limit must not be negative", maxDefaultStock)
	}
	if o.purge && o.dryRun {
		return fmt.Errorf("-purge cannot be combined with -dry-run: it would delete data")
	}
	if err := config.LoadEnv(); err != nil {
		return err
	}
	if o.verifyTarget {
		mongoConfig, err := config.LoadMongoConfig()
		if err != nil {
			return err
		}
		if !o.allowRemote && !isLocalMongo(mongoConfig.URI) {
			return fmt.Errorf("MONGO_URI is not localhost: refusing to migrate or import dev data (use -allow-remote to override)")
		}
		fmt.Println("target database is local")
		return nil
	}
	if o.purge {
		if !o.yes {
			return fmt.Errorf("-purge deletes data: re-run with -yes")
		}
		return withDB(o, purge)
	}
	items, err := loadCatalog(o.catalogPath)
	if err != nil {
		return err
	}
	if o.limit > 0 && o.limit < len(items) {
		items = items[:o.limit]
	}
	baseURL := publicURL()
	if err := validateBaseURL(baseURL); err != nil {
		return err
	}
	p, err := buildPlan(items, baseURL, o.defaultStock, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := validateImages(p.Images, o.assetsPath); err != nil {
		return err
	}
	report(p)
	if o.check {
		fmt.Println("check OK: no database access")
		return nil
	}
	return withDB(o, func(ctx context.Context, db *mongo.Database, o importOptions) error { return importPlan(ctx, db, o, p) })
}

func report(p plan) {
	fmt.Printf("catalog: %d categories, %d products, %d images; skipped %d\n", len(p.Categories), len(p.Products), len(p.Images), len(p.Skipped))
	for _, s := range p.Skipped {
		fmt.Printf("  skipped %s (%s)\n", s.Handle, s.Reason)
	}
	for _, w := range p.Warnings {
		fmt.Println("  warning:", w)
	}
}

func publicURL() string {
	if value := strings.TrimSpace(os.Getenv("RUSTFS_PUBLIC_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	port, bucket := os.Getenv("RUSTFS_API_PORT"), os.Getenv("RUSTFS_BUCKET")
	if port == "" {
		port = "9100"
	}
	if bucket == "" {
		bucket = "aodora-images"
	}
	return "http://localhost:" + port + "/" + bucket
}

func validateBaseURL(base string) error {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("RUSTFS_PUBLIC_URL must be a public HTTP(S) bucket URL")
	}
	return nil
}

const (
	maxImageBytes   = 20 * 1024 * 1024
	maxDefaultStock = 100000 // keeps per-product stock sums far from int64 overflow
)

// validateImages checks every referenced file exists, is non-empty, sane in size and its magic bytes match the extension.
func validateImages(images []imageRef, assets string) error {
	for _, im := range images {
		if filepath.IsAbs(im.File) || !filepath.IsLocal(im.File) || !filepath.IsLocal(im.Handle) {
			return fmt.Errorf("unsafe image path %s/%s", im.Handle, im.File)
		}
		path := filepath.Join(assets, im.Handle, im.File)
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("image %s/%s missing: download the Drive folder into %s: %w", im.Handle, im.File, assets, err)
		}
		head := make([]byte, 12)
		n, _ := io.ReadFull(f, head)
		info, statErr := f.Stat()
		_ = f.Close()
		if statErr != nil {
			return statErr
		}
		if n < 12 || info.Size() > maxImageBytes {
			return fmt.Errorf("image %s/%s is empty, truncated or over 20MB", im.Handle, im.File)
		}
		if !magicMatches(strings.ToLower(filepath.Ext(im.File)), head) {
			return fmt.Errorf("image %s/%s content does not match its extension", im.Handle, im.File)
		}
	}
	return nil
}

func magicMatches(ext string, head []byte) bool {
	switch ext {
	case ".png":
		return bytes.HasPrefix(head, []byte("\x89PNG\r\n\x1a\n"))
	case ".jpg":
		return bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF})
	case ".webp":
		return string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP"
	}
	return false
}

// isLocalMongo reports whether every host in the URI is a loopback host.
func isLocalMongo(uri string) bool {
	rest := uri
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		rest = rest[i+1:]
	}
	if i := strings.IndexAny(rest, "/?"); i >= 0 {
		rest = rest[:i]
	}
	for host := range strings.SplitSeq(rest, ",") {
		host = strings.TrimSpace(host)
		if strings.HasPrefix(host, "[") {
			if i := strings.Index(host, "]"); i >= 0 {
				host = host[:i+1]
			}
		} else if i := strings.LastIndex(host, ":"); i >= 0 {
			host = host[:i]
		}
		switch host {
		case "localhost", "127.0.0.1", "[::1]":
		default:
			return false
		}
	}
	return rest != ""
}

func withDB(o importOptions, fn func(context.Context, *mongo.Database, importOptions) error) error {
	mongoConfig, err := config.LoadMongoConfig()
	if err != nil {
		return err
	}
	if !o.allowRemote && !isLocalMongo(mongoConfig.URI) {
		return fmt.Errorf("MONGO_URI is not localhost: refusing to write dev data (use -allow-remote to override)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := database.OpenMongo(ctx, database.Config{URI: mongoConfig.URI, DB: mongoConfig.DB})
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(context.Background()) }()
	for _, name := range []string{"categories", "products"} {
		names, err := db.DB.ListCollectionNames(ctx, bson.M{"name": name})
		if err != nil {
			return err
		}
		if len(names) == 0 {
			return fmt.Errorf("missing %s collection: run make migrate-up first", name)
		}
	}
	return fn(ctx, db.DB, o)
}

// updatable lists source-owned fields refreshed by -update. Stock, price and version are never in it.
var updatable = []string{"name", "description", "material", "image", "gallery", "images", "colors", "source_url", "seed_source", "source_handle", "collection", "story"}

func importPlan(ctx context.Context, db *mongo.Database, o importOptions, p plan) error {
	categories, products := db.Collection("categories"), db.Collection("products")
	for _, doc := range p.Categories {
		var other bson.M
		err := categories.FindOne(ctx, bson.M{"slug": doc["slug"], "_id": bson.M{"$ne": doc["_id"]}}).Decode(&other)
		if err == nil {
			return fmt.Errorf("category slug %v is already used by another record", doc["slug"])
		} else if err != mongo.ErrNoDocuments {
			return err
		}
	}
	for _, doc := range p.Products {
		var other bson.M
		err := products.FindOne(ctx, bson.M{"slug": doc["slug"], "_id": bson.M{"$ne": doc["_id"]}}).Decode(&other)
		if err == nil {
			return fmt.Errorf("product slug %v is already used by another record", doc["slug"])
		} else if err != mongo.ErrNoDocuments {
			return err
		}
	}
	if err := upsertAll(ctx, categories, "categories", p.Categories, o, false); err != nil {
		return err
	}
	return upsertAll(ctx, products, "products", p.Products, o, o.update)
}

func upsertAll(ctx context.Context, coll *mongo.Collection, label string, docs []bson.M, o importOptions, refresh bool) error {
	inserted, preserved, updated := 0, 0, 0
	for _, doc := range docs {
		if o.dryRun {
			n, err := coll.CountDocuments(ctx, bson.M{"_id": doc["_id"]})
			if err != nil {
				return err
			}
			if n == 0 {
				inserted++
			} else {
				preserved++
			}
			continue
		}
		update := bson.M{"$setOnInsert": doc}
		if refresh {
			set, insert := bson.M{}, bson.M{}
			maps.Copy(insert, doc)
			unset := bson.M{}
			for _, k := range updatable {
				if v, ok := doc[k]; ok {
					set[k] = v
					delete(insert, k)
				} else {
					unset[k] = "" // the source no longer provides it (e.g. material without a label)
				}
			}
			set["updated_at"] = doc["updated_at"]
			delete(insert, "updated_at")
			update = bson.M{"$setOnInsert": insert, "$set": set}
			if len(unset) > 0 {
				update["$unset"] = unset
			}
		}
		res, err := coll.UpdateOne(ctx, bson.M{"_id": doc["_id"]}, update, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("import %s %v: %w", label, doc["slug"], err)
		}
		switch {
		case res.UpsertedCount > 0:
			inserted++
		case refresh && res.ModifiedCount > 0:
			updated++
		default:
			preserved++
		}
	}
	verb := "inserted"
	if o.dryRun {
		verb = "would insert"
	}
	fmt.Printf("%s: %s %d, preserved %d, refreshed %d\n", label, verb, inserted, preserved, updated)
	return nil
}

// purge removes only what this importer created: products with seed_source=sumire-store and emptied sumire-* categories.
func purge(ctx context.Context, db *mongo.Database, _ importOptions) error {
	res, err := db.Collection("products").DeleteMany(ctx, bson.M{"seed_source": seedSource})
	if err != nil {
		return err
	}
	fmt.Printf("products: deleted %d\n", res.DeletedCount)
	deleted := int64(0)
	for _, c := range categories {
		id := stableID("category", c.Key)
		n, err := db.Collection("products").CountDocuments(ctx, bson.M{"category_id": id})
		if err != nil {
			return err
		}
		if n > 0 {
			fmt.Printf("categories: kept %s (still has %d products)\n", slugPrefix+c.Key, n)
			continue
		}
		r, err := db.Collection("categories").DeleteOne(ctx, bson.M{"_id": id, "slug": slugPrefix + c.Key})
		if err != nil {
			return err
		}
		deleted += r.DeletedCount
	}
	fmt.Printf("categories: deleted %d\n", deleted)
	fmt.Println("images in RustFS were not touched")
	return nil
}
