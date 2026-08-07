// Project 3: Storage Drivers
//
// Small, composable interfaces (Getter/Setter/Deleter -> Store), mirroring
// the standard library's io.Reader/io.Writer style. Two backends —
// MemoryStorage and FileStorage — satisfy the same Store interface, but
// only FileStorage also satisfies the OPTIONAL Closer interface, since only
// it has something to flush. A small driver REGISTRY (a dispatch table of
// constructor functions) mirrors how database/sql itself lets you plug in
// different drivers by name.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
)

// --- Small, composable interfaces -----------------------------------

type Getter interface {
	Get(key string) (any, error)
}

type Setter interface {
	Set(key string, value any) error
}

type Deleter interface {
	Delete(key string) error
}

// Store is COMPOSED from three smaller interfaces — same idea as the
// guide's ReadWriter, just with three pieces instead of two.
type Store interface {
	Getter
	Setter
	Deleter
	Keys() []string
}

// Closer is OPTIONAL — not every Store needs cleanup, so it stays separate
// rather than being folded into Store itself.
type Closer interface {
	Close() error
}

// --- MemoryStorage -----------------------------------------------------

// MemoryStorage satisfies Store but NOT Closer — there's nothing to flush,
// since nothing here ever leaves memory.
type MemoryStorage struct {
	data map[string]any
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{data: make(map[string]any)}
}

func (m *MemoryStorage) Get(key string) (any, error) {
	v, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	return v, nil
}

func (m *MemoryStorage) Set(key string, value any) error {
	m.data[key] = value
	return nil
}

func (m *MemoryStorage) Delete(key string) error {
	if _, ok := m.data[key]; !ok {
		return fmt.Errorf("key %q not found", key)
	}
	delete(m.data, key)
	return nil
}

func (m *MemoryStorage) Keys() []string {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// --- FileStorage -----------------------------------------------------

// FileStorage buffers changes in memory and only writes to disk on Close —
// this is exactly WHY it needs to satisfy Closer and MemoryStorage doesn't:
// there's real cleanup work (flushing to disk) that has to happen before
// the program exits, or writes are silently lost.
type FileStorage struct {
	path  string
	data  map[string]any
	dirty bool
}

func NewFileStorage(path string) (*FileStorage, error) {
	fs := &FileStorage{path: path, data: make(map[string]any)}

	existing, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(existing, &fs.data); err != nil {
			return nil, fmt.Errorf("parsing existing file %q: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}
	return fs, nil
}

func (f *FileStorage) Get(key string) (any, error) {
	v, ok := f.data[key]
	if !ok {
		return nil, fmt.Errorf("key %q not found", key)
	}
	return v, nil
}

func (f *FileStorage) Set(key string, value any) error {
	f.data[key] = value
	f.dirty = true
	return nil
}

func (f *FileStorage) Delete(key string) error {
	if _, ok := f.data[key]; !ok {
		return fmt.Errorf("key %q not found", key)
	}
	delete(f.data, key)
	f.dirty = true
	return nil
}

func (f *FileStorage) Keys() []string {
	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Close flushes to disk ONLY if something actually changed — this is what
// makes FileStorage satisfy Closer, on top of Store.
func (f *FileStorage) Close() error {
	if !f.dirty {
		return nil
	}
	bytes, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding data: %w", err)
	}
	if err := os.WriteFile(f.path, bytes, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", f.path, err)
	}
	f.dirty = false
	return nil
}

// closeIfPossible is the OPTIONAL-INTERFACE pattern from the Payment
// Gateway and Notification Service projects, applied to lifecycle
// management: check whether this particular Store also happens to be a
// Closer, and clean up only if so.
func closeIfPossible(s Store) error {
	if c, ok := s.(Closer); ok {
		return c.Close()
	}
	return nil // nothing to do — e.g. MemoryStorage
}

// --- Driver registry -----------------------------------------------------

// driverRegistry is a dispatch table of CONSTRUCTOR functions, keyed by
// name — the same higher-order-function-as-data idea from Module 03's
// Calculator API, applied here to mirror how database/sql lets you plug in
// different drivers ("postgres", "sqlite", ...) by name at runtime.
var driverRegistry = map[string]func(config string) (Store, error){
	"memory": func(config string) (Store, error) {
		return NewMemoryStorage(), nil
	},
	"file": func(config string) (Store, error) {
		return NewFileStorage(config) // config is the file path for this driver
	},
}

func openStore(driver, config string) (Store, error) {
	constructor, ok := driverRegistry[driver]
	if !ok {
		return nil, fmt.Errorf("unknown storage driver %q", driver)
	}
	return constructor(config)
}

// describeValue uses REFLECTION to report each stored value's concrete
// type, useful when a Store's values (typed `any`) could be anything.
func describeValue(v any) string {
	return reflect.TypeOf(v).String()
}

func demo(name string, store Store) {
	fmt.Printf("\n=== %s ===\n", name)

	store.Set("username", "kemi")
	store.Set("loginCount", 17)
	store.Set("preferences", map[string]bool{"darkMode": true})

	for _, key := range store.Keys() {
		v, _ := store.Get(key)
		fmt.Printf("  %-12s = %-25v (%s)\n", key, v, describeValue(v))
	}

	if err := store.Delete("loginCount"); err != nil {
		fmt.Println("  delete error:", err)
	}
	fmt.Println("  after delete, keys:", store.Keys())

	if err := closeIfPossible(store); err != nil {
		fmt.Println("  close error:", err)
	} else {
		fmt.Println("  closeIfPossible: done (no-op if this Store isn't a Closer)")
	}
}

func main() {
	memStore, err := openStore("memory", "")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	demo("MemoryStorage (driver: \"memory\")", memStore)

	fileStore, err := openStore("file", "storage-data.json")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	demo("FileStorage (driver: \"file\")", fileStore)
	fmt.Println("\nCheck storage-data.json in this folder — it now holds the")
	fmt.Println("flushed data, written by FileStorage's Close() method.")
}
