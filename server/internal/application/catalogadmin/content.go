package catalogadmin

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ProductContent is the three editorial fields already present in the MongoDB
// products schema. Nil sections mean that the section is not published.
// This separate write contract cannot modify price, inventory, variants or reviews.
type ProductContent struct {
	Story           *Story  `json:"story"`
	Fabric          *Fabric `json:"fabric"`
	Craft           *Craft  `json:"craft"`
	ExpectedVersion *int64  `json:"expected_version"`
}

type Story struct {
	Eyebrow    string   `json:"eyebrow" bson:"eyebrow"`
	Title      string   `json:"title" bson:"title"`
	Paragraphs []string `json:"paragraphs" bson:"paragraphs"`
	Image      *Image   `json:"image" bson:"image,omitempty"`
}

type ContentItem struct {
	Icon        string `json:"icon,omitempty" bson:"icon,omitempty"`
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
}

type CraftStep struct {
	Title       string `json:"title" bson:"title"`
	Description string `json:"description" bson:"description"`
}

type Fabric struct {
	Eyebrow     string        `json:"eyebrow" bson:"eyebrow"`
	Title       string        `json:"title" bson:"title"`
	Description string        `json:"description" bson:"description"`
	Image       *Image        `json:"image" bson:"image,omitempty"`
	Care        []ContentItem `json:"care" bson:"care"`
}

type Craft struct {
	Eyebrow     string      `json:"eyebrow" bson:"eyebrow"`
	Title       string      `json:"title" bson:"title"`
	Description string      `json:"description" bson:"description"`
	Image       *Image      `json:"image" bson:"image,omitempty"`
	Steps       []CraftStep `json:"steps" bson:"steps"`
}

type ContentRecord struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Version int64   `json:"version"`
	Story   *Story  `json:"story"`
	Fabric  *Fabric `json:"fabric"`
	Craft   *Craft  `json:"craft"`
}

func contentText(text, name string, limit int, required bool) error {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) > limit || strings.ContainsRune(text, 0) || (required && strings.TrimSpace(text) == "") {
		return Invalid{Field: name, Message: "Nội dung trống, không hợp lệ hoặc quá dài"}
	}
	return nil
}

func validateProductContent(v *ProductContent) error {
	if err := expectedVersion(v.ExpectedVersion); err != nil {
		return err
	}
	if story := v.Story; story != nil {
		if err := contentText(story.Eyebrow, "story.eyebrow", 160, false); err != nil {
			return err
		}
		if err := contentText(story.Title, "story.title", 250, false); err != nil {
			return err
		}
		if err := validateImage(story.Image, "story.image"); err != nil {
			return err
		}
		if len(story.Paragraphs) > 30 {
			return Invalid{"story.paragraphs", "Tối đa 30 đoạn kể"}
		}
		for i, p := range story.Paragraphs {
			if err := contentText(p, fmt.Sprintf("story.paragraphs[%d]", i), 3000, true); err != nil {
				return err
			}
		}
		if strings.TrimSpace(story.Title) == "" && strings.TrimSpace(story.Eyebrow) == "" && len(story.Paragraphs) == 0 && story.Image == nil {
			return Invalid{"story", "Dùng null nếu muốn xóa toàn bộ câu chuyện"}
		}
		if story.Paragraphs == nil {
			story.Paragraphs = []string{}
		}
	}
	if fabric := v.Fabric; fabric != nil {
		if err := validateSection(fabric.Eyebrow, fabric.Title, fabric.Description, fabric.Image, "fabric"); err != nil {
			return err
		}
		if len(fabric.Care) > 30 {
			return Invalid{"fabric.care", "Tối đa 30 hướng dẫn bảo quản"}
		}
		for i, item := range fabric.Care {
			if err := contentText(item.Icon, fmt.Sprintf("fabric.care[%d].icon", i), 80, false); err != nil {
				return err
			}
			if err := contentText(item.Title, fmt.Sprintf("fabric.care[%d].title", i), 200, true); err != nil {
				return err
			}
			if err := contentText(item.Description, fmt.Sprintf("fabric.care[%d].description", i), 2000, true); err != nil {
				return err
			}
		}
		if emptySection(fabric.Eyebrow, fabric.Title, fabric.Description, fabric.Image) && len(fabric.Care) == 0 {
			return Invalid{"fabric", "Dùng null nếu muốn xóa phần chất liệu"}
		}
		if fabric.Care == nil {
			fabric.Care = []ContentItem{}
		}
	}
	if craft := v.Craft; craft != nil {
		if err := validateSection(craft.Eyebrow, craft.Title, craft.Description, craft.Image, "craft"); err != nil {
			return err
		}
		if len(craft.Steps) > 30 {
			return Invalid{"craft.steps", "Tối đa 30 bước chế tác"}
		}
		for i, item := range craft.Steps {
			if err := contentText(item.Title, fmt.Sprintf("craft.steps[%d].title", i), 200, true); err != nil {
				return err
			}
			if err := contentText(item.Description, fmt.Sprintf("craft.steps[%d].description", i), 2000, true); err != nil {
				return err
			}
		}
		if emptySection(craft.Eyebrow, craft.Title, craft.Description, craft.Image) && len(craft.Steps) == 0 {
			return Invalid{"craft", "Dùng null nếu muốn xóa phần chế tác"}
		}
		if craft.Steps == nil {
			craft.Steps = []CraftStep{}
		}
	}
	return nil
}

func emptySection(eyebrow, title, desc string, image *Image) bool {
	return strings.TrimSpace(eyebrow) == "" && strings.TrimSpace(title) == "" && strings.TrimSpace(desc) == "" && image == nil
}
func validateSection(eyebrow, title, desc string, image *Image, prefix string) error {
	for _, item := range []struct {
		value, field string
		max          int
	}{
		{eyebrow, prefix + ".eyebrow", 160}, {title, prefix + ".title", 250}, {desc, prefix + ".description", 4000},
	} {
		if err := contentText(item.value, item.field, item.max, false); err != nil {
			return err
		}
	}
	return validateImage(image, prefix+".image")
}

func (s *Service) ProductContent(ctx context.Context, id string) (ContentRecord, error) {
	return s.repo.ProductContent(ctx, id)
}
func (s *Service) UpdateProductContent(ctx context.Context, id string, data ProductContent) (ContentRecord, error) {
	if err := validateProductContent(&data); err != nil {
		return ContentRecord{}, err
	}
	return s.repo.UpdateProductContent(ctx, id, data)
}
