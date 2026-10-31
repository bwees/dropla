package storage

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

func TestSyncingUploadWritesAcrossSyncBoundaries(t *testing.T) {
	const syncEvery = 8
	cases := map[string]int{
		"shorter than one sync": syncEvery - 3,
		"exact multiple":        syncEvery * 3,
		"partial last sync":     syncEvery*3 + 5,
	}

	for name, size := range cases {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := SyncingStore{FileStore: filestore.New(t.TempDir()), syncEvery: syncEvery}
			data := bytes.Repeat([]byte("dropla"), size)[:size]

			upload, err := store.NewUpload(ctx, tusd.FileInfo{Size: int64(size)})
			if err != nil {
				t.Fatalf("new upload: %v", err)
			}

			n, err := upload.WriteChunk(ctx, 0, bytes.NewReader(data))
			if err != nil {
				t.Fatalf("write chunk: %v", err)
			}
			if n != int64(size) {
				t.Fatalf("wrote %d bytes, want %d", n, size)
			}

			info, err := upload.GetInfo(ctx)
			if err != nil {
				t.Fatalf("get info: %v", err)
			}
			got, err := os.ReadFile(info.Storage[filestore.StorageKeyPath])
			if err != nil {
				t.Fatalf("read upload: %v", err)
			}
			if !bytes.Equal(got, data) {
				t.Fatalf("upload content differs from what was sent")
			}

			reopened, err := store.GetUpload(ctx, info.ID)
			if err != nil {
				t.Fatalf("get upload: %v", err)
			}
			reopenedInfo, err := reopened.GetInfo(ctx)
			if err != nil {
				t.Fatalf("get reopened info: %v", err)
			}
			if reopenedInfo.Offset != int64(size) {
				t.Fatalf("offset %d, want %d", reopenedInfo.Offset, size)
			}
		})
	}
}

func TestSyncingStoreTerminatesUpload(t *testing.T) {
	ctx := context.Background()
	store := NewSyncingStore(t.TempDir())

	upload, err := store.NewUpload(ctx, tusd.FileInfo{Size: 1})
	if err != nil {
		t.Fatalf("new upload: %v", err)
	}
	info, err := upload.GetInfo(ctx)
	if err != nil {
		t.Fatalf("get info: %v", err)
	}

	if err := store.AsTerminatableUpload(upload).Terminate(ctx); err != nil {
		t.Fatalf("terminate: %v", err)
	}
	if _, err := os.Stat(info.Storage[filestore.StorageKeyPath]); !os.IsNotExist(err) {
		t.Fatalf("upload file still exists after terminate")
	}
}
