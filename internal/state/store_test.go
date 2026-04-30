package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreUpsertAndGet(t *testing.T) {
	store := &Store{path: filepath.Join(t.TempDir(), "state.json")}
	key := FileKey("sample.txt", 10, 20)
	input := FileState{Path: "sample.txt", Size: 10, ModTime: 20, Offset: 5, Encoding: "utf8"}

	if err := store.Upsert(key, input); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.Get(key)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected saved file state")
	}
	if got.Offset != 5 || got.Encoding != "utf8" {
		t.Fatalf("unexpected state: %#v", got)
	}
}

func TestNewStoreMigratesLegacyState(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("AppData", configDir)
	legacyDir := filepath.Join(configDir, "bf-reader")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatal(err)
	}
	legacyData := []byte(`{"files":{"sample|1|2":{"path":"sample","size":1,"modTime":2,"offset":9,"encoding":"utf8","bookmarks":[]}}}`)
	if err := os.WriteFile(filepath.Join(legacyDir, "state.json"), legacyData, 0644); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore("bigtext-reader", "bf-reader")
	if err != nil {
		t.Fatal(err)
	}
	migrated, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(migrated) != string(legacyData) {
		t.Fatalf("unexpected migrated state: %s", migrated)
	}
}
