package fromimmich

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/coverageTester"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/immichfs"
)

type FromImmich struct {
	flags *FromImmichFlags
	// client *app.Client
	ifs *immichfs.ImmichFS
	ic  *filenames.InfoCollector

	mustFetchAlbums bool // True if we need to fetch the asset's albums in 2nd step
	errCount        int  // Count the number of errors, to stop after 5
}

func NewFromImmich(ctx context.Context, app *app.Application, jnl *fileevent.Recorder, flags *FromImmichFlags) (*FromImmich, error) {
	client := &flags.client
	err := client.Initialize(ctx, app)
	if err != nil {
		return nil, err
	}
	err = client.Open(ctx)
	if err != nil {
		return nil, err
	}

	ifs := immichfs.NewImmichFS(ctx, flags.client.Server, client.Immich)
	f := FromImmich{
		flags: flags,
		ifs:   ifs,
		ic:    filenames.NewInfoCollector(time.Local, client.Immich.SupportedMedia()),
	}
	return &f, nil
}

func (f *FromImmich) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)
		var err error
		switch {
		case len(f.flags.Albums) > 0:
			err = f.getAssetsFromAlbums(ctx, gOut)
		default:
			err = f.getAssets(ctx, gOut)
		}
		if err != nil {
			f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting Immich assets: %v", err))
		}
	}()
	return gOut
}

const timeFormat = "2006-01-02T15:04:05.000Z"

func (f *FromImmich) getAssets(ctx context.Context, grpChan chan *assets.Group) error {
	query := immich.SearchMetadataQuery{
		Make:  f.flags.Make,
		Model: f.flags.Model,
		// WithExif:     true,
		WithArchived: f.flags.WithArchived,
	}

	f.mustFetchAlbums = true
	if f.flags.DateRange.IsSet() {
		query.TakenAfter = f.flags.DateRange.After.Format(timeFormat)
		query.TakenBefore = f.flags.DateRange.Before.Format(timeFormat)
	}

	return f.flags.client.Immich.GetAllAssetsWithFilter(ctx, &query, func(a *immich.Asset) error {
		if f.flags.Favorite && !a.IsFavorite {
			return nil
		}
		if !f.flags.WithTrashed && a.IsTrashed {
			return nil
		}
		return f.filterAsset(ctx, a, grpChan)
	})
}

func (f *FromImmich) getAssetsFromAlbums(ctx context.Context, grpChan chan *assets.Group) error {
	f.mustFetchAlbums = false

	assets := map[string]*immich.Asset{} // List of assets to get by ID

	albums, err := f.flags.client.Immich.GetAllAlbums(ctx)
	if err != nil {
		return f.logError(err)
	}
	for _, album := range albums {
		for _, albumName := range f.flags.Albums {
			if album.Title == albumName {
				al, err := f.flags.client.Immich.GetAlbumInfo(ctx, album.ID, false)
				if err != nil {
					return f.logError(err)
				}
				for _, a := range al.Assets {
					if _, ok := assets[a.ID]; !ok {
						a.Albums = append(a.Albums, immich.AlbumSimplified{
							AlbumName: album.Title,
						})
						assets[a.ID] = a
					} else {
						assets[a.ID].Albums = append(assets[a.ID].Albums, immich.AlbumSimplified{
							AlbumName: album.Title,
						})
					}
				}
			}
		}
	}

	for _, a := range assets {
		err = f.filterAsset(ctx, a, grpChan)
		if err != nil {
			return f.logError(err)
		}
	}
	return nil
}

func (f *FromImmich) filterAsset(ctx context.Context, a *immich.Asset, grpChan chan *assets.Group) error {
	var err error

	fmt.Println(f)
	// Checks if favourited asset and flag is favourite

	// Checks if favourited asset and flag is favourite
	if f.flags.Favorite && !a.IsFavorite {
		coverageTester.WriteUniqueLine("filterAsset - Branch 1/16 Covered")
		coverageTester.WriteUniqueLine("Branch 1")
		return nil
	}

	// Probably filtering based on if photo is put in the bin
	// Probably filtering based on if photo is put in the bin
	if !f.flags.WithTrashed && a.IsTrashed {
		coverageTester.WriteUniqueLine("filterAsset - Branch 2/16 Covered")
		coverageTester.WriteUniqueLine("Branch 2")
		return nil
	}

	// Refactor the album section
	// Refactor the album section
	albums := immich.AlbumsFromAlbumSimplified(a.Albums)

	// Some filter set up in getAsset to determine is albums much be fetched later.
	// Some filter set up in getAsset to determine is albums much be fetched later.
	if f.mustFetchAlbums && len(albums) == 0 {
		coverageTester.WriteUniqueLine("filterAsset - Branch 3/16 Covered")
		coverageTester.WriteUniqueLine("Branch 3")
		albums, err = f.flags.client.Immich.GetAssetAlbums(ctx, a.ID)
		if err != nil {
			coverageTester.WriteUniqueLine("filterAsset - Branch 4/16 Covered")
			coverageTester.WriteUniqueLine("Branch 4")
			return f.logError(err)
		}
	}

	// Checks if the asset is present in any albums by f.flags.Albums.

	// Checks if the asset is present in any albums by f.flags.Albums.
	if len(f.flags.Albums) > 0 && len(albums) > 0 {
		coverageTester.WriteUniqueLine("filterAsset - Branch 5/16 Covered")
		coverageTester.WriteUniqueLine("Branch 5")
		keepMe := false
		newAlbumList := []assets.Album{}
		for _, album := range f.flags.Albums {
			coverageTester.WriteUniqueLine("filterAsset - Branch 6/16 Covered")
			coverageTester.WriteUniqueLine("Branch 6")
			for _, aAlbum := range albums {
				coverageTester.WriteUniqueLine("filterAsset - Branch 7/16 Covered")
				coverageTester.WriteUniqueLine("Branch 7")
				if album == aAlbum.Title {
					coverageTester.WriteUniqueLine("filterAsset - Branch 8/16 Covered")
					coverageTester.WriteUniqueLine("Branch 8")
					keepMe = true
					newAlbumList = append(newAlbumList, aAlbum)
				}
			}
		}
		if !keepMe {
			coverageTester.WriteUniqueLine("filterAsset - Branch 9/16 Covered")
			coverageTester.WriteUniqueLine("Branch 9")
			return nil
		}
		albums = newAlbumList
	}

	// Some information are missing in the metadata result,
	// so we need to get the asset details
	a, err = f.flags.client.Immich.GetAssetInfo(ctx, a.ID)
	if err != nil {
		coverageTester.WriteUniqueLine("filterAsset - Branch 10/16 Covered")
		coverageTester.WriteUniqueLine("Branch 10")
		return f.logError(err)
	}
	asset := a.AsAsset()
	asset.SetNameInfo(f.ic.GetInfo(asset.OriginalFileName))
	asset.File = fshelper.FSName(f.ifs, a.ID)

	asset.FromApplication = &assets.Metadata{
		FileName:    a.OriginalFileName,
		Latitude:    a.ExifInfo.Latitude,
		Longitude:   a.ExifInfo.Longitude,
		Description: a.ExifInfo.Description,
		DateTaken:   a.ExifInfo.DateTimeOriginal.Time,
		Trashed:     a.IsTrashed,
		Archived:    a.IsArchived,
		Favorited:   a.IsFavorite,
		Rating:      byte(a.Rating),
		Albums:      albums,
		Tags:        asset.Tags,
	}

	// Filter on rating, unsure what kind of rating though.
	// Filter on rating, unsure what kind of rating though.
	if f.flags.MinimalRating > 0 && a.Rating < f.flags.MinimalRating {
		coverageTester.WriteUniqueLine("filterAsset - Branch 11/16 Covered")
		coverageTester.WriteUniqueLine("Branch 11")
		return nil
	}

	// Filter asset on set date range.
	// Filter asset on set date range.
	if f.flags.DateRange.IsSet() {
		coverageTester.WriteUniqueLine("filterAsset - Branch 12/16 Covered")
		coverageTester.WriteUniqueLine("Branch 12")
		if asset.CaptureDate.Before(f.flags.DateRange.After) || asset.CaptureDate.After(f.flags.DateRange.Before) {
			coverageTester.WriteUniqueLine("filterAsset - Branch 13/16 Covered")
			coverageTester.WriteUniqueLine("Branch 13")
			return nil
		}
	}

	// This part is used to channel the asset
	// This part is used to channel the asset
	g := assets.NewGroup(assets.GroupByNone, asset)
	select {
	case grpChan <- g:
		coverageTester.WriteUniqueLine("filterAsset - Branch 14/16 Covered")
		coverageTester.WriteUniqueLine("Branch 14")
	case <-ctx.Done():
		coverageTester.WriteUniqueLine("filterAsset - Branch 15/16 Covered")
		coverageTester.WriteUniqueLine("Branch 15")
		return ctx.Err()
	}
	coverageTester.WriteUniqueLine("filterAsset - Branch 16/16 Covered")
	coverageTester.WriteUniqueLine("Branch 16")
	return nil
}

func (f *FromImmich) logError(err error) error {
	f.flags.client.ClientLog.Error(fmt.Sprintf("Error while getting Immich assets: %v", err))
	f.errCount++
	if f.errCount > 5 {
		err := errors.New("too many errors, aborting")
		f.flags.client.ClientLog.Error(err.Error())
		return err
	}
	return nil
}
