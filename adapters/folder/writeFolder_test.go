package folder

import (
	"context"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// TestWriteAsset_mini calls WriteAsset with an empty asset to test the function without having it crash
func TestWriteAsset_mini(t *testing.T) {
	writer := &LocalAssetWriter{}

	asset := &assets.Asset{
		File: fshelper.FSAndName{},
	}

	_ = writer.WriteAsset(context.Background(), asset)
}

// TestWriteAsset_cancelledContext checks that WriteAsset exits early when the context is cancelled
func TestWriteAsset_cancelledContext(t *testing.T) {
	writer := &LocalAssetWriter{}

	asset := &assets.Asset{}

	// create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// WriteAsset detects cancelled context and returns error
	err := writer.WriteAsset(ctx, asset)
	if err == nil {
		t.Errorf("Expected error due to cancelled context")
	}
}

// TestWriteAsset_samename tests WriteAsset when two jpg's are uploaded with the same name
// It only works in theory as afero does not support writing but it gives good coverage
// The branches that this test covers that the other two does not is 6, 7, 8 and 11
// Due to not working fully it's outcommented.

/*func TestWriteAsset_samename(t *testing.T) {

	outputFS := afero.NewMemMapFs()

	iofs := afero.NewIOFS(outputFS)

	writer := &LocalAssetWriter{
		WriteToFS:  iofs,
		createdDir: make(map[string]struct{}),
	}

	createFakeAsset := func(name string, content string) *assets.Asset {

		return &assets.Asset{
			OriginalFileName: name,
			File:             fshelper.FSName(fakeInputFS{content: content}, name),
		}

	}

	// upload first photo
	asset1 := createFakeAsset("photo.jpg", "fake image content 1")
	if err := writer.WriteAsset(context.Background(), asset1); err != nil {
		t.Fatalf("failed to write the first asset %v", err)
	}

	// upload second photo
	asset2 := createFakeAsset("photo.jpg", "fake image content 2")
	if err := writer.WriteAsset(context.Background(), asset2); err != nil {
		t.Fatalf("failed to write the second asset %v", err)
	}

	if _, err := outputFS.Open("photo.jpg"); err != nil {
		t.Errorf("Expected photo.jpg to exsist but it does not %v", err)
	}

	if _, err := outputFS.Open("photo~1.jpg"); err != nil {
		t.Errorf("Expected photo~1.jpg to exist due to duplicates, but it does not %v", err)
	}
}

type fakeInputFS struct {
	content string
}

func (fs fakeInputFS) Open(name string) (fs.File, error) {
	return &mockFile{Reader: bytes.NewReader([]byte(fs.content))}, nil
}

type mockFile struct {
	*bytes.Reader
}

func (m *mockFile) Stat() (fs.FileInfo, error) { return nil, nil }
func (m *mockFile) Close() error               { return nil }
*/
