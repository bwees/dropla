package storage

import (
	"context"
	"errors"
	"io"
	"os"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

// defaultSyncEvery bounds how much of an upload can sit in the page cache
// before it is flushed. Unflushed pages count against the container's memory
// limit, and on NFS they drain slower than a client can send them.
const defaultSyncEvery int64 = 32 << 20

// SyncingStore is tusd's filestore, except a PATCH is flushed to disk every
// syncEvery bytes instead of only being buffered.
type SyncingStore struct {
	filestore.FileStore
	syncEvery int64
}

func NewSyncingStore(path string) SyncingStore {
	return SyncingStore{FileStore: filestore.New(path), syncEvery: defaultSyncEvery}
}

// UseIn leaves out the Concater: filestore's concatenation needs its own
// upload type, and no dropla client uses it.
func (s SyncingStore) UseIn(composer *tusd.StoreComposer) {
	composer.UseCore(s)
	composer.UseTerminater(s)
	composer.UseLengthDeferrer(s)
	composer.UseContentServer(s)
}

func (s SyncingStore) NewUpload(ctx context.Context, info tusd.FileInfo) (tusd.Upload, error) {
	upload, err := s.FileStore.NewUpload(ctx, info)
	if err != nil {
		return nil, err
	}
	return s.wrap(ctx, upload)
}

func (s SyncingStore) GetUpload(ctx context.Context, id string) (tusd.Upload, error) {
	upload, err := s.FileStore.GetUpload(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.wrap(ctx, upload)
}

func (s SyncingStore) AsTerminatableUpload(upload tusd.Upload) tusd.TerminatableUpload {
	return s.FileStore.AsTerminatableUpload(upload.(*syncingUpload).Upload)
}

func (s SyncingStore) AsLengthDeclarableUpload(upload tusd.Upload) tusd.LengthDeclarableUpload {
	return s.FileStore.AsLengthDeclarableUpload(upload.(*syncingUpload).Upload)
}

func (s SyncingStore) AsServableUpload(upload tusd.Upload) tusd.ServableUpload {
	return s.FileStore.AsServableUpload(upload.(*syncingUpload).Upload)
}

func (s SyncingStore) wrap(ctx context.Context, upload tusd.Upload) (tusd.Upload, error) {
	info, err := upload.GetInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &syncingUpload{
		Upload:    upload,
		path:      info.Storage[filestore.StorageKeyPath],
		syncEvery: s.syncEvery,
	}, nil
}

type syncingUpload struct {
	tusd.Upload
	path      string
	syncEvery int64
}

// WriteChunk also slows the client to the speed of the disk, because the
// flush blocks the request.
func (u *syncingUpload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	var written int64
	for {
		n, err := u.Upload.WriteChunk(ctx, offset+written, io.LimitReader(src, u.syncEvery))
		written += n
		if err != nil {
			return written, err
		}
		if err := u.sync(); err != nil {
			return written, err
		}
		if n < u.syncEvery {
			return written, nil
		}
	}
}

func (u *syncingUpload) sync() error {
	file, err := os.Open(u.path)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}
