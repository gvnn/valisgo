package storage_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"valisgo/internal/storage"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/memblob"
)

func TestBlobStorage(t *testing.T) {
	ctx := context.Background()
	bucket, err := blob.OpenBucket(ctx, "mem://")
	if err != nil {
		t.Fatalf("failed to open mem bucket: %v", err)
	}
	defer bucket.Close()

	store := storage.NewBlobStorage(bucket)

	key := "test-file.txt"
	content := []byte("hello world")

	err = store.Put(ctx, key, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read from Get: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("expected %q, got %q", content, got)
	}

	err = store.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = store.Get(ctx, key)
	if err == nil {
		t.Error("expected error getting deleted file, got nil")
	}
}
