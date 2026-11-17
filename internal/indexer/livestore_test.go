package indexer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/nofollow"
)

func TestIndexRefusesLiveStoreBeforeBuilding(t *testing.T) {
	type refusal struct {
		name    string
		prepare func(t *testing.T, root string) (storeDir string, check func(t *testing.T))
		want    error
	}
	for _, tc := range []refusal{
		{"a directory that is not a store", func(t *testing.T, root string) (string, func(*testing.T)) {
			src := filepath.Join(root, "src")
			main := filepath.Join(src, "main.go")
			writeFile(t, main, "package main\n")
			return src, func(t *testing.T) {
				entries, err := os.ReadDir(src)
				if err != nil || len(entries) != 1 {
					t.Fatalf("%s holds %d entries (err %v), want only main.go", src, len(entries), err)
				}
				if data, err := os.ReadFile(main); err != nil || string(data) != "package main\n" {
					t.Fatalf("main.go = %q, %v, want it unchanged", data, err)
				}
			}
		}, graphstore.ErrNotStore},
		{"a store that holds a link", func(t *testing.T, root string) (string, func(*testing.T)) {
			storeDir := filepath.Join(root, ".codegraph", "store")
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			outside := filepath.Join(t.TempDir(), "outside")
			writeFile(t, outside, "keep me\n")
			if err := os.Symlink(outside, filepath.Join(storeDir, "OPTIONS-999999")); err != nil {
				t.Fatal(err)
			}
			return storeDir, func(t *testing.T) {
				if data, err := os.ReadFile(outside); err != nil || string(data) != "keep me\n" {
					t.Fatalf("outside file = %q, %v, want it unchanged", data, err)
				}
			}
		}, nofollow.ErrSymlink},
	} {
		for name, index := range map[string]func(root, storeDir string) (Stats, error){
			"Run":  func(root, storeDir string) (Stats, error) { return Run(root, storeDir, Options{}) },
			"Sync": func(root, storeDir string) (Stats, error) { return Sync(root, storeDir, Options{}) },
		} {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				root := goTree(t, 20, false)
				storeDir, check := tc.prepare(t, root)

				stats, err := index(root, storeDir)
				if !errors.Is(err, tc.want) {
					t.Fatalf("%s = %v, want %v", name, err, tc.want)
				}
				if stats.Nodes != 0 {
					t.Fatalf("%s built %d nodes before it refused the store", name, stats.Nodes)
				}
				check(t)
			})
		}
	}
}
