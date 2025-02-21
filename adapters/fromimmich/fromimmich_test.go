package fromimmich

import (
	"context"
	"fmt"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

func TestFilterAsset_Favourite(t *testing.T) {

	// Mocks a minimal FromImmich instance
	flags := &FromImmichFlags{
		Favorite:      true,
		MinimalRating: 4,
	}

	// Mocks the FromImmich object
	fromimmich := &FromImmich{
		flags: flags,
	}

	// Create a mock asset
	asset := &immich.Asset{
		ID:               "12345",
		Rating:           3,
		IsFavorite:       false,
		OriginalFileName: "test.jpg",
	}

	// Creates a mock channel
	grpChan := make(chan *assets.Group)

	// Calls the function under test
	err := fromimmich.filterAsset(context.Background(), asset, grpChan)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	fmt.Println("Test completed!")
}

func TestFilterAsset_Rating(t *testing.T) {

	// Mocks a minimal FromImmich instance
	flags := &FromImmichFlags{
		Favorite:      false,
		MinimalRating: 4,
		WithTrashed:   false,
	}

	// Mocks the FromImmich object
	fromimmich := &FromImmich{
		flags: flags,
	}

	// Create a mock asset
	asset := &immich.Asset{
		ID:               "12345",
		IsTrashed:        true,
		Rating:           3,
		IsFavorite:       false,
		OriginalFileName: "test.jpg",
	}

	// Creates a mock channel
	grpChan := make(chan *assets.Group)

	// Calls the function under test
	err := fromimmich.filterAsset(context.Background(), asset, grpChan)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
}
