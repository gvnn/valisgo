package store_test

import (
	"testing"

	"valisgo/internal/store"
)

func TestRegistryStore_Init(t *testing.T) {
	s := store.NewRegistryStore(nil)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
}
