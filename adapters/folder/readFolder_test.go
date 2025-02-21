package folder

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/kr/pretty"
	"github.com/psanford/memfs"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/namematcher"
)

type inMemFS struct {
	*memfs.FS
	name string
	err  error
	ic   *filenames.InfoCollector
}

func newInMemFS(name string, ic *filenames.InfoCollector) *inMemFS { // nolint: unparam
	return &inMemFS{
		name: name,
		FS:   memfs.New(),
		ic:   ic,
	}
}

func (mfs inMemFS) Name() string {
	return mfs.name
}

func (mfs *inMemFS) addFile(name string, _ time.Time) *inMemFS {
	if mfs.err != nil {
		return mfs
	}
	dir := path.Dir(name)
	base := path.Base(name)
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0o777))
	i := mfs.ic.GetInfo(base)
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, *(*[]byte)(unsafe.Pointer(&i)), 0o777))
	return mfs
}

func TestInMemLocalAssets(t *testing.T) {
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	tc := []struct {
		name           string
		fsys           []fs.FS
		flags          ImportFolderOptions
		expectedFiles  []string
		expectedCounts []int64
		expectedAlbums map[string][]string
	}{
		{
			name: "easy",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InfoCollector:  ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0),
			},
			expectedFiles:  []string{"root_01.jpg"},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 1).Value(),
		},
		{
			name: "recursive",
			flags: ImportFolderOptions{
				InfoCollector:  ic,
				SupportedMedia: filetypes.DefaultSupportedMedia,
				Recursive:      true,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0),
			},
			expectedFiles:  []string{"root_01.jpg", "photos/photo_01.jpg"},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 2).Value(),
		},
		{
			name: "non-recursive",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InfoCollector:  ic,
				Recursive:      false,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0),
			},
			expectedFiles:  []string{"root_01.jpg"},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 1).Value(),
		},

		{
			name: "banned files",
			flags: ImportFolderOptions{
				BannedFiles:    namematcher.MustList(`@eaDir`, `.@__thumb`, `SYNOFILE_THUMB_*.*`, "BLOG/", "Database/", `._*.*`, `._*.*`),
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InclusionFlags: cliflags.InclusionFlags{},
				InfoCollector:  ic,
				Recursive:      true,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0).
					addFile("@eaDir/thb1.jpg", t0).
					addFile("photos/SYNOFILE_THUMB_0001.jpg", t0).
					addFile("photos/summer 2023/.@__thumb/thb2.jpg", t0).
					addFile("BLOG/blog.jpg", t0).
					addFile("Project/Database/database_01.jpg", t0).
					addFile("photos/database_01.jpg", t0).
					addFile("mac/image.JPG", t0).
					addFile("mac/._image.JPG", t0).
					addFile("mac/image.JPG", t0).
					addFile("mac/._image.JPG", t0),
			},
			expectedFiles: []string{
				"root_01.jpg",
				"photos/photo_01.jpg",
				"photos/photo_02.cr3",
				"photos/photo_03.jpg",
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
				"photos/summer 2023/20230801-003.cr3",
				"photos/database_01.jpg",
				"mac/image.JPG",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 9).
				Set(fileevent.DiscoveredDiscarded, 6).Value(),
		},
		{
			name: "excluded extensions",
			flags: ImportFolderOptions{
				BannedFiles:    namematcher.MustList(`@eaDir/`, `.@__thumb`, `SYNOFILE_THUMB_*.*`),
				SupportedMedia: filetypes.DefaultSupportedMedia,

				InclusionFlags: cliflags.InclusionFlags{
					ExcludedExtensions: cliflags.ExtensionList{".cr3"},
				},
				Recursive:     true,
				InfoCollector: ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0).
					addFile("@eaDir/thb1.jpg", t0).
					addFile("photos/SYNOFILE_THUMB_0001.jpg", t0).
					addFile("photos/summer 2023/.@__thumb/thb2.jpg", t0),
			},
			expectedFiles: []string{
				"root_01.jpg",
				"photos/photo_01.jpg",
				"photos/photo_03.jpg",
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 7).
				Set(fileevent.DiscoveredDiscarded, 5).Value(),
		},
		{
			name: "included extensions",
			flags: ImportFolderOptions{
				BannedFiles:    namematcher.MustList(`@eaDir/`, `.@__thumb`, `SYNOFILE_THUMB_*.*`),
				SupportedMedia: filetypes.DefaultSupportedMedia,

				InclusionFlags: cliflags.InclusionFlags{
					IncludedExtensions: cliflags.ExtensionList{".cr3"},
				},
				Recursive:     true,
				InfoCollector: ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0).
					addFile("@eaDir/thb1.jpg", t0).
					addFile("photos/SYNOFILE_THUMB_0001.jpg", t0).
					addFile("photos/summer 2023/.@__thumb/thb2.jpg", t0),
			},
			expectedFiles: []string{
				"photos/photo_02.cr3",
				"photos/summer 2023/20230801-003.cr3",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 7).
				Set(fileevent.DiscoveredDiscarded, 8).Value(),
		},
		{
			name: "motion picture",
			flags: ImportFolderOptions{
				BannedFiles:    namematcher.MustList(`@eaDir/`, `.@__thumb`, `SYNOFILE_THUMB_*.*`),
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InclusionFlags: cliflags.InclusionFlags{},
				Recursive:      true,
				InfoCollector:  ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("motion/nomotion.MP4", t0).
					addFile("motion/PXL_20210102_221126856.MP~2", t0).
					addFile("motion/PXL_20210102_221126856.MP~2.jpg", t0).
					addFile("motion/PXL_20210102_221126856.MP.jpg", t0).
					addFile("motion/PXL_20210102_221126856.MP", t0).
					addFile("motion/20231227_152817.jpg", t0).
					addFile("motion/20231227_152817.MP4", t0).
					addFile("motion/MVIMG_20180418_113218", t0).
					addFile("motion/MVIMG_20180418_113218.jpg", t0),
			},
			expectedFiles: []string{
				"motion/PXL_20210102_221126856.MP.jpg",
				"motion/PXL_20210102_221126856.MP~2.jpg",
				"motion/20231227_152817.jpg", "motion/20231227_152817.MP4",
				"motion/nomotion.MP4",
				"motion/MVIMG_20180418_113218.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).
				Set(fileevent.DiscoveredVideo, 2).
				Set(fileevent.DiscoveredUseless, 3).Value(),
		},

		{
			name: "date in range, use name",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,

				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange(time.Local, "2023-08"),
				},
				Recursive:            true,
				TZ:                   time.Local,
				TakeDateFromFilename: true,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0),
			},
			expectedFiles: []string{
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
				"photos/summer 2023/20230801-003.cr3",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 7).
				Set(fileevent.DiscoveredDiscarded, 4).
				Set(fileevent.INFO, 7).Value(),
		},

		{
			name: "path as album name",
			flags: ImportFolderOptions{
				SupportedMedia:         filetypes.DefaultSupportedMedia,
				UsePathAsAlbumName:     FolderModePath,
				AlbumNamePathSeparator: " ¤ ",
				InclusionFlags:         cliflags.InclusionFlags{},
				Recursive:              true,
				InfoCollector:          ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0),
			},
			expectedFiles: []string{
				"root_01.jpg",
				"photos/photo_01.jpg",
				"photos/photo_02.cr3",
				"photos/photo_03.jpg",
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
				"photos/summer 2023/20230801-003.cr3",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 7).Value(),
			expectedAlbums: map[string][]string{
				"MemFS":                        {"root_01.jpg"},
				"MemFS ¤ photos":               {"photos/photo_01.jpg", "photos/photo_02.cr3", "photos/photo_03.jpg"},
				"MemFS ¤ photos ¤ summer 2023": {"photos/summer 2023/20230801-001.jpg", "photos/summer 2023/20230801-002.jpg", "photos/summer 2023/20230801-003.cr3"},
			},
		},

		{
			name: "folder as album name",
			flags: ImportFolderOptions{
				SupportedMedia:         filetypes.DefaultSupportedMedia,
				UsePathAsAlbumName:     FolderModeFolder,
				AlbumNamePathSeparator: " ¤ ",
				InclusionFlags:         cliflags.InclusionFlags{},
				Recursive:              true,
				InfoCollector:          ic,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/photo_02.cr3", t0).
					addFile("photos/photo_03.jpg", t0).
					addFile("photos/summer 2023/20230801-001.jpg", t0).
					addFile("photos/summer 2023/20230801-002.jpg", t0).
					addFile("photos/summer 2023/20230801-003.cr3", t0),
			},
			expectedFiles: []string{
				"root_01.jpg",
				"photos/photo_01.jpg",
				"photos/photo_02.cr3",
				"photos/photo_03.jpg",
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
				"photos/summer 2023/20230801-003.cr3",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 7).Value(),
			expectedAlbums: map[string][]string{
				"MemFS":       {"root_01.jpg"},
				"photos":      {"photos/photo_01.jpg", "photos/photo_02.cr3", "photos/photo_03.jpg"},
				"summer 2023": {"photos/summer 2023/20230801-001.jpg", "photos/summer 2023/20230801-002.jpg", "photos/summer 2023/20230801-003.cr3"},
			},
		},
	}

	logFile := configuration.DefaultLogFile()
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			log := app.Log{
				File:  logFile,
				Level: "INFO",
			}
			err := log.OpenLogFile()
			if err != nil {
				t.Error(err)
				return
			}
			log.Logger.Info("\n\n\ntest case: " + c.name)
			recorder := fileevent.NewRecorder(log.Logger)
			b, err := NewLocalFiles(ctx, recorder, &c.flags, c.fsys...)
			if err != nil {
				t.Error(err)
			}

			groupChan := b.Browse(ctx)

			results := []string{}
			albums := map[string][]string{}

			for g := range groupChan {
				if err = g.Validate(); err != nil {
					t.Error(err)
					return
				}
				for _, a := range g.Assets {
					results = append(results, a.File.Name())
					if len(c.expectedAlbums) > 0 {
						for _, album := range a.Albums {
							albums[album.Title] = append(albums[album.Title], a.File.Name())
						}
					}
				}
			}

			sort.Strings(c.expectedFiles)
			sort.Strings(results)

			if !reflect.DeepEqual(results, c.expectedFiles) {
				t.Errorf("file list difference\n")
				pretty.Ldiff(t, c.expectedFiles, results)
			}
			if !reflect.DeepEqual(recorder.GetCounts(), c.expectedCounts) {
				t.Errorf("counters difference\n")
				pretty.Ldiff(t, c.expectedCounts, recorder.GetCounts())
			}
			if c.expectedAlbums != nil {
				compareAlbums(t, albums, c.expectedAlbums)
			}
		})
	}
}

func TestInMemLocalAssetsWithTags(t *testing.T) {
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	tc := []struct {
		name  string
		fsys  []fs.FS
		flags ImportFolderOptions
		want  map[string][]string
	}{
		{
			name: "tags",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InfoCollector:  ic,
				Recursive:      true,
				Tags:           []string{"tag1", "tag2/subtag2"},
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0),
			},
			want: map[string][]string{
				"root_01.jpg":         {"tag1", "tag2/subtag2"},
				"photos/photo_01.jpg": {"tag1", "tag2/subtag2"},
			},
		},
		{
			name: "folder as tags",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InfoCollector:  ic,
				Recursive:      true,
				FolderAsTags:   true,
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/summer/photo_02.jpg", t0),
			},
			want: map[string][]string{
				"root_01.jpg":                {"MemFS"},
				"photos/photo_01.jpg":        {"MemFS/photos"},
				"photos/summer/photo_02.jpg": {"MemFS/photos/summer"},
			},
		},
		{
			name: "folder as tags and a tag",
			flags: ImportFolderOptions{
				SupportedMedia: filetypes.DefaultSupportedMedia,
				InfoCollector:  ic,
				Recursive:      true,
				FolderAsTags:   true,
				Tags:           []string{"tag1"},
			},
			fsys: []fs.FS{
				newInMemFS("MemFS", ic).
					addFile("root_01.jpg", t0).
					addFile("photos/photo_01.jpg", t0).
					addFile("photos/summer/photo_02.jpg", t0),
			},
			want: map[string][]string{
				"root_01.jpg":                {"tag1", "MemFS"},
				"photos/photo_01.jpg":        {"tag1", "MemFS/photos"},
				"photos/summer/photo_02.jpg": {"tag1", "MemFS/photos/summer"},
			},
		},
	}

	logFile := configuration.DefaultLogFile()
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			log := app.Log{
				File:  logFile,
				Level: "INFO",
			}
			err := log.OpenLogFile()
			if err != nil {
				t.Error(err)
				return
			}
			log.Logger.Info("\n\n\ntest case: " + c.name)
			recorder := fileevent.NewRecorder(log.Logger)
			b, err := NewLocalFiles(ctx, recorder, &c.flags, c.fsys...)
			if err != nil {
				t.Error(err)
			}

			groupChan := b.Browse(ctx)

			got := map[string][]string{}

			for g := range groupChan {
				if err = g.Validate(); err != nil {
					t.Error(err)
					return
				}
				for _, a := range g.Assets {
					tags := []string{}
					for _, tag := range a.Tags {
						tags = append(tags, tag.Value)
					}

					got[a.File.Name()] = tags
				}
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("tags difference\n")
				pretty.Ldiff(t, c.want, got)
			}
		})
	}
}

// Testing to read JSON from filestructure
func TestParseDir_WithJSON(t *testing.T) {
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	ctx := context.Background()
	logFile := configuration.DefaultLogFile()
	log := app.Log{
		File:  logFile,
		Level: "INFO",
	}
	err := log.OpenLogFile()
	if err != nil {
		t.Error(err)
		return
	}
	recorder := fileevent.NewRecorder(log.Logger)
	
	gOut := make(chan *assets.Group)
	defer close(gOut)
	// Start a goroutine to consume the gOut channel
	go func() {
		for group := range gOut {
				t.Logf("Received group: %+v", group)
		}
	}()

	fsys := newInMemFS("MemFS", ic).
		addFile("root_01.jpg", t0).
		addFile("photos/photo_01.jpg", t0).
		addFile("photos/photo_01.json", t0).
		addFile("photos/summer/photo_02.jpg", t0)
	flags := &ImportFolderOptions{
		UsePathAsAlbumName: FolderModePath,
		InfoCollector:      ic,
		SupportedMedia:     filetypes.DefaultSupportedMedia,
	}
	la, err := NewLocalFiles(ctx, recorder, flags, fsys)

	if err != nil {
		t.Errorf("Error, %v", err)
		return
	}

	err = la.parseDir(ctx, fsys, "photos", gOut)

	if err != nil {
		t.Errorf("Error, %v", err)
	}
}

// Test parseDir with canceled context
func TestParseDir_ClosedContext(t *testing.T) {
	t0 := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	ic := filenames.NewInfoCollector(time.Local, filetypes.DefaultSupportedMedia)
	ctx, cancel := context.WithCancel(context.Background())
	logFile := configuration.DefaultLogFile()
	log := app.Log{
		File:  logFile,
		Level: "INFO",
	}
	err := log.OpenLogFile()
	if err != nil {
		t.Error(err)
		cancel()
		return
	}
	recorder := fileevent.NewRecorder(log.Logger)
	
	gOut := make(chan *assets.Group)
	defer close(gOut)
	// Start a goroutine to consume the gOut channel
	go func() {
		for group := range gOut {
				t.Logf("Received group: %+v", group)
		}
	}()

	fsys := newInMemFS("MemFS", ic).
		addFile("root_01.jpg", t0).
		addFile("photos/photo_01.jpg", t0).
		addFile("photos/summer/photo_02.jpg", t0)
	flags := &ImportFolderOptions{
		UsePathAsAlbumName: FolderModePath,
		InfoCollector:      ic,
		SupportedMedia:     filetypes.DefaultSupportedMedia,
	}
	la, err := NewLocalFiles(ctx, recorder, flags, fsys)

	if err != nil {
		t.Errorf("Error, %v", err)
		cancel()
		return
	}

	cancel()

	err = la.parseDir(ctx, fsys, "photos", gOut)

	if err == nil {
		t.Errorf("Error, %v", err)
	}
}

func compareAlbums(t *testing.T, a, b map[string][]string) {
	a = sortAlbum(a)
	b = sortAlbum(b)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("album list difference\n")
		pretty.Ldiff(t, a, b)
	}
}

func sortAlbum(a map[string][]string) map[string][]string {
	for k := range a {
		sort.Strings(a[k])
	}
	return a
}

func TestNewLocalFiles_ConflictingAlbumFlags(t *testing.T) {
	ctx := context.Background()
	recorder := &fileevent.Recorder{}
	flags := &ImportFolderOptions{
		ImportIntoAlbum:    "TestAlbum",
		UsePathAsAlbumName: FolderModePath,
	}

	la, err := NewLocalFiles(ctx, recorder, flags)

	if err == nil || !strings.Contains(err.Error(), "cannot use both --into-album and --folder-as-album") {
		t.Errorf("Expected conflict error, got: %v", err)
	}
	if la != nil {
		t.Errorf("Expected nil la due to error, got: %v", la)
	}
}

func TestNewLocalFiles_PicasaAlbumEnabled(t *testing.T) {
	ctx := context.Background()
	recorder := &fileevent.Recorder{}
	flags := &ImportFolderOptions{
		PicasaAlbum: true,
	}

	la, err := NewLocalFiles(ctx, recorder, flags)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if la == nil {
		t.Errorf("Expected non-nil la, got nil")
	}
	if la != nil && la.picasaAlbums == nil {
		t.Errorf("Expected la.picasaAlbums to be initialized, but it was nil")
	}
}

func TestNewLocalFiles_SessionTagEnabled(t *testing.T) {
	ctx := context.Background()
	recorder := &fileevent.Recorder{}
	flags := &ImportFolderOptions{
		SessionTag: true,
	}

	la, err := NewLocalFiles(ctx, recorder, flags)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedPrefix := "{immich-go}/"
	if la.flags.session == "" || !strings.HasPrefix(la.flags.session, expectedPrefix) {
		t.Errorf("Expected session to start with %q, but got %q", expectedPrefix, la.flags.session)
	}

	_, err = time.Parse("2006-01-02 15:04:05", strings.TrimPrefix(la.flags.session, expectedPrefix))
	if err != nil {
		t.Errorf("Session timestamp format is incorrect: %v", err)
	}
}

func TestNewLocalFiles_EpsonFastFotoAndBurstEnabled(t *testing.T) {
	ctx := context.Background()
	recorder := &fileevent.Recorder{}
	flags := &ImportFolderOptions{
		ManageEpsonFastFoto: true,
		ManageBurst:         filters.BurstKeepJPEG,
	}

	la, err := NewLocalFiles(ctx, recorder, flags)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(la.groupers) < 2 {
		t.Errorf("Expected at least 2 groupers in la.groupers, but got %d", len(la.groupers))
	}
}
