package query

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FilesOptions configures Engine.Files' browse of the indexed file
// structure (QRY-07). The zero value browses every indexed file, in the
// default "flat" format, with no depth limit.
type FilesOptions struct {
	// Pattern is a shell glob (path/filepath.Match semantics, matched
	// against the full forward-slashed file path) that narrows the
	// result set. Empty matches every file.
	Pattern string

	// Filter narrows results to files whose Language exactly matches.
	// Empty applies no language filter.
	Filter string

	// Depth caps directory nesting: a file whose path has Depth or more
	// "/" separators is excluded (Depth=1 means root-level files only,
	// Depth=2 allows one level of nesting, and so on). 0 (the zero
	// value) means unlimited — deliberately NOT clampDepth's "0 means a
	// small BFS-safe default" convention (that convention exists to
	// bound unbounded graph traversal cost; Files is a bounded scan over
	// an already-enumerated file set, so "browse everything" is the
	// useful default). Negative values and values above MaxDepth are
	// rejected outright (validateFilesDepth) rather than silently
	// clamped, matching validateLimit's "reject absurd input" pattern
	// (T-03-05-DoS, reusing the existing validate-helper convention).
	Depth int

	// Format selects the projection: "flat" (default, or when empty)
	// returns FilesResult.Files, a sorted list of FileEntry records.
	// "tree" returns FilesResult.Tree, the same entries grouped into a
	// nested directory structure. Any other value is rejected.
	Format string
}

// FileEntry is one browsed file's projection: identity plus the
// aggregate symbol/edge counts already stored on its schema.File record
// (no additional graph work per entry).
type FileEntry struct {
	Path      string `json:"path"`
	Language  string `json:"language"`
	NodeCount int64  `json:"nodeCount"`
	EdgeCount int64  `json:"edgeCount"`
}

// FileTreeNode is one node of the "tree" format's nested directory
// projection. Directory nodes carry Children and no Path/Language; file
// (leaf) nodes carry Path/Language and no Children.
type FileTreeNode struct {
	Name     string          `json:"name"`
	IsDir    bool            `json:"isDir"`
	Path     string          `json:"path,omitempty"`
	Language string          `json:"language,omitempty"`
	Children []*FileTreeNode `json:"children,omitempty"`
}

// FilesResult is Engine.Files' return shape: exactly one of Files (flat
// format) or Tree (tree format) is populated, per Format.
type FilesResult struct {
	Format string          `json:"format"`
	Files  []FileEntry     `json:"files,omitempty"`
	Tree   []*FileTreeNode `json:"tree,omitempty"`
}

// validateFilesDepth rejects a negative depth or a depth above MaxDepth
// (T-03-05-DoS — "reject absurd depths, reuse validate helpers"). 0 is
// accepted and means "unlimited" (see FilesOptions.Depth doc).
func validateFilesDepth(n int) error {
	if n < 0 {
		return fmt.Errorf("query: depth %d must be non-negative", n)
	}
	if n > MaxDepth {
		return fmt.Errorf("query: depth %d exceeds maximum %d", n, MaxDepth)
	}
	return nil
}

// Files browses the indexed file structure from the graph (QRY-07): it
// reads e.reader.IterateFiles() only — never os.ReadDir or any other live
// filesystem walk — so the result reflects the frozen point-in-time graph
// even if the working tree has since changed underneath it. pattern
// narrows by glob match on the full path, filter narrows by exact
// Language match, depth bounds directory nesting, and format selects the
// flat-list vs. nested-tree projection. The returned set is additionally
// capped at MaxLimit entries (T-03-05-DoS, "cap the returned set").
func (e *Engine) Files(opts FilesOptions) (FilesResult, error) {
	if err := validateFilesDepth(opts.Depth); err != nil {
		return FilesResult{}, err
	}
	format := opts.Format
	if format == "" {
		format = "flat"
	}
	if format != "flat" && format != "tree" {
		return FilesResult{}, fmt.Errorf("query: unknown files format %q — allowed: flat, tree", format)
	}
	if opts.Pattern != "" {
		if _, err := filepath.Match(opts.Pattern, "sanity-check"); err != nil {
			return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
		}
	}

	it, err := e.reader.IterateFiles()
	if err != nil {
		return FilesResult{}, err
	}
	defer it.Close()

	entries := []FileEntry{}
	for it.Next() {
		f := it.File()
		p := filepath.ToSlash(f.Path)

		if opts.Filter != "" && f.Language != opts.Filter {
			continue
		}
		if opts.Pattern != "" {
			matched, err := filepath.Match(opts.Pattern, p)
			if err != nil {
				return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
			}
			if !matched {
				continue
			}
		}
		if opts.Depth > 0 && strings.Count(p, "/") >= opts.Depth {
			continue
		}

		entries = append(entries, FileEntry{
			Path:      p,
			Language:  f.Language,
			NodeCount: f.NodeCount,
			EdgeCount: f.EdgeCount,
		})
	}
	if err := it.Err(); err != nil {
		return FilesResult{}, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	if len(entries) > MaxLimit {
		entries = entries[:MaxLimit]
	}

	result := FilesResult{Format: format}
	if format == "tree" {
		result.Tree = buildFileTree(entries)
	} else {
		result.Files = entries
	}
	return result, nil
}

// buildFileTree groups already-filtered/sorted entries into a nested
// directory structure by splitting each Path on "/". Directories are
// synthesized on demand (they carry no FileEntry of their own — the
// graph only stores file records, D-03) and sorted directories-first,
// then alphabetically, at every level.
func buildFileTree(entries []FileEntry) []*FileTreeNode {
	root := &FileTreeNode{IsDir: true}
	for _, entry := range entries {
		segments := strings.Split(entry.Path, "/")
		cur := root
		for i, seg := range segments {
			isFile := i == len(segments)-1
			var child *FileTreeNode
			for _, c := range cur.Children {
				if c.Name == seg && c.IsDir == !isFile {
					child = c
					break
				}
			}
			if child == nil {
				child = &FileTreeNode{Name: seg, IsDir: !isFile}
				if isFile {
					child.Path = entry.Path
					child.Language = entry.Language
				}
				cur.Children = append(cur.Children, child)
			}
			cur = child
		}
	}
	sortFileTree(root.Children)
	return root.Children
}

// sortFileTree orders nodes directories-first then alphabetically by
// name, recursively, so the tree projection is deterministic across
// calls (matching this package's general "--json output must be
// reproducible" convention, D-06).
func sortFileTree(nodes []*FileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})
	for _, n := range nodes {
		if n.IsDir {
			sortFileTree(n.Children)
		}
	}
}

// MarshalFilesJSON renders a FilesResult as --json output. There is no
// golden fixture for files (D-07a) — this is this plan's own designed
// shape, marshaled as a thin passthrough matching search.go/traverse.go's
// Marshal* convention.
func MarshalFilesJSON(r FilesResult) ([]byte, error) {
	return json.Marshal(r)
}
