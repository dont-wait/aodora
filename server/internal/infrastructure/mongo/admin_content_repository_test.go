package mongo

import (
	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"testing"
)

func TestContentMutationIsIsolatedFromOtherProductFields(t *testing.T) {
	version := int64(0)
	input := app.ProductContent{
		Story:           &app.Story{Title: "Mẫu A", Paragraphs: []string{"Đoạn một"}},
		ExpectedVersion: &version,
	}
	update := contentMutation(input)
	set := asMap(update["$set"])
	unset := asMap(update["$unset"])
	for _, key := range []string{"price", "stock", "variants", "sizes", "reviews", "rating", "name", "slug", "material", "image", "gallery"} {
		if _, ok := set[key]; ok {
			t.Fatalf("content mutation rewrites %s", key)
		}
		if _, ok := unset[key]; ok {
			t.Fatalf("content mutation unsets %s", key)
		}
	}
	if _, ok := set["story"]; !ok {
		t.Fatal("missing story update")
	}
	if unset["fabric"] != "" || unset["craft"] != "" {
		t.Fatalf("cleared sections not removed: %v", unset)
	}
	if asMap(update["$inc"])["version"] != int64(1) {
		t.Fatal("missing optimistic version bump")
	}
	filter := withMetadataVersion(bson.NewObjectID(), 0)
	if _, ok := filter["$or"]; !ok {
		t.Fatal("legacy versions not handled")
	}
}
