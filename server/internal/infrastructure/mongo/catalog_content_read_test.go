package mongo

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestProductDetailPreservesPerProductEditorialOrder(t *testing.T) {
	first := bson.M{
		"_id": bson.NewObjectID(), "price": int64(500000), "slug": "mau-a",
		"story":  bson.M{"title": "A", "paragraphs": bson.A{"một", "hai", "ba"}},
		"fabric": bson.M{"title": "Lụa A", "care": bson.A{bson.M{"title": "Giặt", "description": "Giặt tay"}}},
		"craft":  bson.M{"steps": bson.A{bson.M{"title": "Cắt", "description": "Cắt vải"}}},
	}
	second := bson.M{"_id": bson.NewObjectID(), "price": int64(400000), "slug": "mau-b"}
	a, err := productFromBSON(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := productFromBSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if a.Slug != "mau-a" || a.Story == nil || len(a.Story.Paragraphs) != 3 || a.Story.Paragraphs[2] != "ba" {
		t.Fatalf("story order lost: %+v", a.Story)
	}
	if a.Fabric == nil || a.Fabric.Care[0].Title != "Giặt" || a.Craft == nil || a.Craft.Steps[0].Title != "Cắt" {
		t.Fatal("fabric/craft lost")
	}
	if b.Story != nil || b.Fabric != nil || b.Craft != nil {
		t.Fatal("another product inherited the story")
	}
}
