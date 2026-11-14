package catalogadmin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContentValidationRejectsBrokenStoryAndUnsafeImage(t *testing.T) {
	n := int64(0)
	service := NewService(&fakeRepo{})
	valid := ProductContent{ExpectedVersion: &n, Story: &Story{
		Title: "Câu chuyện", Paragraphs: []string{"Đoạn một", "Đoạn hai", "Đoạn ba"},
	}, Fabric: &Fabric{Title: "Lụa", Care: []ContentItem{{Title: "Giặt", Description: "Giặt tay"}}},
		Craft: &Craft{Steps: []CraftStep{{Title: "Cắt", Description: "Cắt theo rập"}}},
	}
	if _, err := service.UpdateProductContent(context.Background(), "id", valid); err != nil {
		t.Fatal(err)
	}
	missing := valid
	missing.ExpectedVersion = nil
	if _, err := service.UpdateProductContent(context.Background(), "id", missing); err == nil {
		t.Fatal("accepted missing concurrency version")
	}
	invalid := valid
	invalid.Story = &Story{Paragraphs: []string{"   "}}
	if _, err := service.UpdateProductContent(context.Background(), "id", invalid); err == nil {
		t.Fatal("accepted blank paragraph")
	}
	invalid = valid
	invalid.Story = &Story{Title: "Tên", Image: &Image{Src: "javascript:alert(1)", Alt: "ảnh"}}
	if _, err := service.UpdateProductContent(context.Background(), "id", invalid); err == nil {
		t.Fatal("accepted unsafe image URL")
	}
	invalid = valid
	invalid.Fabric = &Fabric{Title: "Lụa", Care: []ContentItem{{Title: "Giặt"}}}
	if _, err := service.UpdateProductContent(context.Background(), "id", invalid); err == nil {
		t.Fatal("accepted missing care description")
	}
	invalid = valid
	invalid.Craft = &Craft{Steps: []CraftStep{{Title: "", Description: "Thêu"}}}
	if _, err := service.UpdateProductContent(context.Background(), "id", invalid); err == nil {
		t.Fatal("accepted nameless craft step")
	}
	invalid = valid
	invalid.Story = &Story{Title: strings.Repeat("a", 300)}
	if _, err := service.UpdateProductContent(context.Background(), "id", invalid); err == nil {
		t.Fatal("accepted oversized title")
	}
}

func TestContentSectionsCanBeCleared(t *testing.T) {
	version := int64(2)
	record, err := NewService(&fakeRepo{}).UpdateProductContent(context.Background(), "id", ProductContent{ExpectedVersion: &version})
	if err != nil || record.Story != nil || record.Fabric != nil || record.Craft != nil || record.Version != 3 {
		t.Fatalf("clearing content failed: %#v %v", record, err)
	}
}

func TestContentErrorsRemainTyped(t *testing.T) {
	err := validateProductContent(&ProductContent{})
	var invalid Invalid
	if !errors.As(err, &invalid) || invalid.Field != "expected_version" {
		t.Fatalf("wanted missing version error, got %v", err)
	}
}
