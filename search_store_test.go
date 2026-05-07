package main

import (
	"errors"
	"os"
	"testing"

	"bigtext-reader/internal/reader"
)

func TestSearchHitStoreAppendGetWindow(t *testing.T) {
	store, err := newSearchHitStore()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Remove()

	refs := []reader.SearchHitRef{
		{Index: 0, Offset: 10, ByteLength: 3, LineStart: 0, LineEnd: 20, LineNumber: 1},
		{Index: 1, Offset: 30, ByteLength: 4, LineStart: 21, LineEnd: 40, LineNumber: 2},
		{Index: 2, Offset: 50, ByteLength: 5, LineStart: 41, LineEnd: 70, LineNumber: 3},
	}
	for _, ref := range refs {
		if err := store.Append(ref); err != nil {
			t.Fatal(err)
		}
	}
	if store.Count() != len(refs) {
		t.Fatalf("unexpected count: %d", store.Count())
	}
	second, err := store.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if second != refs[1] {
		t.Fatalf("unexpected second ref: %#v", second)
	}
	window, err := store.Window(1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(window) != 2 || window[0] != refs[1] || window[1] != refs[2] {
		t.Fatalf("unexpected window: %#v", window)
	}
}

func TestSearchHitStoreWindowBounds(t *testing.T) {
	store, err := newSearchHitStore()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Remove()

	if err := store.Append(reader.SearchHitRef{Index: 0, Offset: 10}); err != nil {
		t.Fatal(err)
	}
	window, err := store.Window(100, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(window) != 0 {
		t.Fatalf("expected empty window, got %#v", window)
	}
	window, err = store.Window(-10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(window) != 1 || window[0].Index != 0 {
		t.Fatalf("unexpected negative offset window: %#v", window)
	}
}

func TestSearchHitStoreRemoveAndClosedAppend(t *testing.T) {
	store, err := newSearchHitStore()
	if err != nil {
		t.Fatal(err)
	}
	path := store.path
	if err := store.Remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temp file removed, got %v", err)
	}
	if err := store.Append(reader.SearchHitRef{}); !errors.Is(err, errSearchHitStoreClosed) {
		t.Fatalf("expected closed error, got %v", err)
	}
}
