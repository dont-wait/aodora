package mongo

import (
	"context"
	"errors"
	"time"

	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Only the existing products.story/fabric/craft fields are read or written;
// full product PUTs, SKUs and aggregate stock remain owned by PD-03.
type storedProductContent struct {
	ID      bson.ObjectID `bson:"_id"`
	Name    string        `bson:"name"`
	Version int64         `bson:"version"`
	Story   *app.Story    `bson:"story"`
	Fabric  *app.Fabric   `bson:"fabric"`
	Craft   *app.Craft    `bson:"craft"`
}

func (doc storedProductContent) response() app.ContentRecord {
	return app.ContentRecord{
		ID: doc.ID.Hex(), Name: doc.Name, Version: doc.Version,
		Story: doc.Story, Fabric: doc.Fabric, Craft: doc.Craft,
	}
}

func contentProjection() bson.M {
	return bson.M{"_id": 1, "name": 1, "version": 1, "story": 1, "fabric": 1, "craft": 1}
}

func (r *AdminCatalogRepository) ProductContent(ctx context.Context, id string) (app.ContentRecord, error) {
	itemID, err := oid(id)
	if err != nil {
		return app.ContentRecord{}, app.Invalid{Field: "id", Message: "Mã sản phẩm không hợp lệ"}
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	var doc storedProductContent
	err = r.products.FindOne(ctx, bson.M{"_id": itemID}, options.FindOne().SetProjection(contentProjection())).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.ContentRecord{}, app.ErrNotFound
	}
	if err != nil {
		return app.ContentRecord{}, err
	}
	return doc.response(), nil
}

func contentMutation(v app.ProductContent) bson.M {
	set := bson.M{"updated_at": time.Now().UTC()}
	unset := bson.M{}
	if v.Story == nil {
		unset["story"] = ""
	} else {
		set["story"] = v.Story
	}
	if v.Fabric == nil {
		unset["fabric"] = ""
	} else {
		set["fabric"] = v.Fabric
	}
	if v.Craft == nil {
		unset["craft"] = ""
	} else {
		set["craft"] = v.Craft
	}
	update := bson.M{"$set": set, "$inc": bson.M{"version": int64(1)}}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	return update
}

func (r *AdminCatalogRepository) UpdateProductContent(ctx context.Context, id string, v app.ProductContent) (app.ContentRecord, error) {
	itemID, err := oid(id)
	if err != nil {
		return app.ContentRecord{}, app.Invalid{Field: "id", Message: "Mã sản phẩm không hợp lệ"}
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	var doc storedProductContent
	err = r.products.FindOneAndUpdate(ctx,
		withMetadataVersion(itemID, *v.ExpectedVersion),
		contentMutation(v),
		options.FindOneAndUpdate().SetReturnDocument(options.After).SetProjection(contentProjection()),
	).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		if verifyErr := r.missingOrConflict(ctx, r.products, itemID); verifyErr != nil {
			return app.ContentRecord{}, verifyErr
		}
	}
	if err != nil {
		return app.ContentRecord{}, err
	}
	return doc.response(), nil
}
