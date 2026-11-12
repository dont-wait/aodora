// Package catalog contains the public read models for Aodora's product catalog.
// No customer, cart or order information belongs in these models.
package catalog

import (
	"encoding/json"
	"time"
)

type Image struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

type Label struct {
	Label   string `json:"label"`
	Variant string `json:"variant"`
}

type Color struct {
	Name string `json:"name,omitempty"`
	Hex  string `json:"hex"`
}

type Category struct {
	ID          string `json:"id"`
	Key         string `json:"key,omitempty"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	IsActive    bool   `json:"-"`
	SortOrder   int64  `json:"sort_order"`
	Image       *Image `json:"image,omitempty"`
	Badge       *Label `json:"badge,omitempty"`
}

type Variant struct {
	SKU       string `json:"sku"`
	Color     string `json:"color"`
	Size      string `json:"size"`
	Stock     int64  `json:"stock"`
	Available bool   `json:"available"`
}

type Accessory struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Price       json.Number `json:"price"`
	Image       *Image      `json:"image,omitempty"`
	Tag         string      `json:"tag,omitempty"`
	TagVariant  string      `json:"tag_variant,omitempty"`
}

type TrustItem struct {
	Icon        string `json:"icon,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type ContentItem struct {
	Icon        string `json:"icon,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Story struct {
	Eyebrow    string   `json:"eyebrow,omitempty"`
	Title      string   `json:"title,omitempty"`
	Paragraphs []string `json:"paragraphs"`
	Image      *Image   `json:"image,omitempty"`
}

type Fabric struct {
	Eyebrow     string        `json:"eyebrow,omitempty"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Image       *Image        `json:"image,omitempty"`
	Care        []ContentItem `json:"care"`
}

type Craft struct {
	Eyebrow     string        `json:"eyebrow,omitempty"`
	Title       string        `json:"title,omitempty"`
	Description string        `json:"description,omitempty"`
	Image       *Image        `json:"image,omitempty"`
	Steps       []ContentItem `json:"steps"`
}

type ReviewSummary struct {
	Count   int64   `json:"count"`
	Average float64 `json:"average"`
}

// Product is the allowlisted public detail. Stock is informational only; the
// checkout owner must re-read and atomically validate stock on purchase.
type Product struct {
	ID             string        `json:"id"`
	CategoryID     string        `json:"category_id"`
	Slug           string        `json:"slug"`
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	Material       string        `json:"material,omitempty"`
	Price          json.Number   `json:"price"`
	OldPrice       *json.Number  `json:"old_price,omitempty"`
	CompareAtPrice *json.Number  `json:"compare_at_price,omitempty"`
	Stock          int64         `json:"stock"`
	SKU            string        `json:"sku,omitempty"`
	Variants       []Variant     `json:"variants"`
	Images         []string      `json:"images"`
	Image          *Image        `json:"image,omitempty"`
	Gallery        []Image       `json:"gallery"`
	Badge          *Label        `json:"badge,omitempty"`
	Tag            *Label        `json:"tag,omitempty"`
	Colors         []Color       `json:"colors"`
	ColorNote      string        `json:"color_note,omitempty"`
	Sizes          []string      `json:"sizes"`
	Bespoke        bool          `json:"bespoke"`
	ReadyAvailable bool          `json:"ready_available"`
	CTA            string        `json:"cta,omitempty"`
	Bundle         string        `json:"bundle,omitempty"`
	Collection     string        `json:"collection,omitempty"`
	Discount       int64         `json:"discount,omitempty"`
	Availability   string        `json:"availability,omitempty"`
	Accessories    []Accessory   `json:"accessories"`
	Trust          []TrustItem   `json:"trust"`
	Story          *Story        `json:"story,omitempty"`
	Fabric         *Fabric       `json:"fabric,omitempty"`
	Craft          *Craft        `json:"craft,omitempty"`
	Reviews        ReviewSummary `json:"reviews"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type ProductSummary struct {
	ID             string       `json:"id"`
	CategoryID     string       `json:"category_id"`
	Slug           string       `json:"slug"`
	Name           string       `json:"name"`
	Description    string       `json:"description,omitempty"`
	Material       string       `json:"material,omitempty"`
	Price          json.Number  `json:"price"`
	OldPrice       *json.Number `json:"old_price,omitempty"`
	CompareAtPrice *json.Number `json:"compare_at_price,omitempty"`
	Image          *Image       `json:"image,omitempty"`
	Badge          *Label       `json:"badge,omitempty"`
	Tag            *Label       `json:"tag,omitempty"`
	Colors         []Color      `json:"colors"`
	Sizes          []string     `json:"sizes"`
	Bespoke        bool         `json:"bespoke"`
	ReadyAvailable bool         `json:"ready_available"`
	Stock          int64        `json:"stock"`
	CreatedAt      time.Time    `json:"created_at"`
}

func (p *Product) SetAvailability() {
	p.ReadyAvailable = false
	for i := range p.Variants {
		p.Variants[i].Available = p.Stock > 0 && p.Variants[i].Stock > 0
		if p.Variants[i].Available {
			p.ReadyAvailable = true
		}
	}
	if len(p.Variants) == 0 && p.SKU != "" && p.Stock > 0 {
		p.ReadyAvailable = true
	}
}

func (p Product) Summary() ProductSummary {
	return ProductSummary{
		ID: p.ID, CategoryID: p.CategoryID, Slug: p.Slug,
		Name: p.Name, Description: p.Description, Material: p.Material,
		Price: p.Price, OldPrice: p.OldPrice, CompareAtPrice: p.CompareAtPrice,
		Image: p.Image, Badge: p.Badge, Tag: p.Tag, Colors: p.Colors,
		Sizes: p.Sizes, Bespoke: p.Bespoke, ReadyAvailable: p.ReadyAvailable,
		Stock: p.Stock, CreatedAt: p.CreatedAt,
	}
}

type Filter struct {
	Search   string
	Category string
	MinPrice *int64
	MaxPrice *int64
	Color    string
	Size     string
	Material string
	Sort     string
	Page     int64
	PageSize int64
}

type ProductPage struct {
	Data       []ProductSummary `json:"data"`
	Total      int64            `json:"total"`
	Page       int64            `json:"page"`
	PageSize   int64            `json:"page_size"`
	TotalPages int64            `json:"total_pages"`
}
