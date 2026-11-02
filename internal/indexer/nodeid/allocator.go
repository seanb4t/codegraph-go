package nodeid

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// IDFields is a per-file extraction result. RewriteIDs replaces each node
// id that the result holds with rewrite(id).
type IDFields interface {
	RewriteIDs(rewrite func(id string) string)
}

// ParamKey is what a parameter contributes to a signature key: its type
// as written, or "" when the language or the author wrote none, and
// whether it is optional, has a default, or is variadic.
type ParamKey struct {
	Type     string
	Optional bool
	Variadic bool
}

// SignatureKey builds the signature key of a function or method from its
// parameters, under one rule for every language (design D1 of the
// index-invariants change): the ordered parameter types as written, `_`
// for an untyped parameter, a `?` suffix for an optional parameter or one
// with a default, and a `...` suffix for a variadic one. Parameter names
// are left out. A function without parameters keys as "()".
func SignatureKey(params []ParamKey) string {
	parts := make([]string, len(params))
	for i, p := range params {
		t := strings.Join(strings.Fields(p.Type), " ")
		if t == "" {
			t = "_"
		}
		if p.Optional {
			t += "?"
		}
		if p.Variadic {
			t += "..."
		}
		parts[i] = t
	}
	return "(" + strings.Join(parts, ",") + ")"
}

// DeclID is the id of a declaration with a signature key: the hash of
// kind, qualified name, file path and key. A declaration without a
// signature, such as a type or a constant, has an empty key and gets
// NodeID, so its id is the same as before signature keys existed.
func DeclID(kind, qualifiedName, filePath, signatureKey string) string {
	if signatureKey == "" {
		return NodeID(kind, qualifiedName, filePath)
	}
	return hashSegments(kind, kind, qualifiedName, filePath, signatureKey)
}

// OrdinalID is the id of the n-th declaration (n of 2 or more) that
// shares a kind, qualified name, file path and signature key with an
// earlier one in the file: the segments of DeclID plus the decimal
// ordinal.
func OrdinalID(kind, qualifiedName, filePath, signatureKey string, n int) string {
	ord := strconv.Itoa(n)
	if signatureKey == "" {
		return hashSegments(kind, kind, qualifiedName, filePath, ord)
	}
	return hashSegments(kind, kind, qualifiedName, filePath, signatureKey, ord)
}

func hashSegments(kind string, segs ...string) string {
	size := len(segs) * binary.MaxVarintLen64
	for _, s := range segs {
		size += len(s)
	}
	buf := make([]byte, 0, size)
	for _, s := range segs {
		buf = appendSegment(buf, s)
	}
	sum := sha256.Sum256(buf)
	return kind + ":" + hex.EncodeToString(sum[:])[:32]
}

// Allocator gives each declaration occurrence of one file its own node id
// (design D1 of the index-invariants change). An extractor declares every
// occurrence with Declare while it walks the file, in any order, and uses
// the provisional id it gets back for the node and for every edge that
// the declaration's body emits. Finalize then numbers the occurrences of
// each (kind, qualified name, signature key) in source order and rewrites
// the provisional ids to final ones.
type Allocator struct {
	filePath string
	decls    []declaration
	seen     map[string]bool
}

type declaration struct {
	kind, qualifiedName, signatureKey string
	startByte                         uint
	provisional                       string
}

// NewAllocator returns an Allocator for the file at filePath.
func NewAllocator(filePath string) *Allocator {
	return &Allocator{filePath: filePath, seen: make(map[string]bool)}
}

// Declare records a declaration of kind, qualifiedName and signatureKey
// that starts at startByte, and returns its provisional id. Declaring the
// same occurrence twice returns the same id.
func (a *Allocator) Declare(kind, qualifiedName, signatureKey string, startByte uint) string {
	prov := DeclID(kind, qualifiedName, a.filePath, signatureKey) + "@" + strconv.FormatUint(uint64(startByte), 10)
	if !a.seen[prov] {
		a.seen[prov] = true
		a.decls = append(a.decls, declaration{kind, qualifiedName, signatureKey, startByte, prov})
	}
	return prov
}

// Finalize computes the final id of each declared occurrence and rewrites
// the provisional ids in result. For each (kind, qualified name, signature
// key), the occurrence with the lowest start byte gets DeclID, and the
// n-th one gets OrdinalID with ordinal n.
//
// After the rewrite, Finalize scans every exported field of result. It
// returns an error if a provisional id is left anywhere, for example in a
// field that RewriteIDs does not know, so that a provisional id never
// reaches the store.
func (a *Allocator) Finalize(result IDFields) error {
	sorted := make([]declaration, len(a.decls))
	copy(sorted, a.decls)
	sort.SliceStable(sorted, func(i, j int) bool {
		x, y := sorted[i], sorted[j]
		if x.kind != y.kind {
			return x.kind < y.kind
		}
		if x.qualifiedName != y.qualifiedName {
			return x.qualifiedName < y.qualifiedName
		}
		if x.signatureKey != y.signatureKey {
			return x.signatureKey < y.signatureKey
		}
		return x.startByte < y.startByte
	})
	final := make(map[string]string, len(sorted))
	n := 0
	for i, d := range sorted {
		if i > 0 && sorted[i-1].kind == d.kind && sorted[i-1].qualifiedName == d.qualifiedName && sorted[i-1].signatureKey == d.signatureKey {
			n++
		} else {
			n = 1
		}
		if n == 1 {
			final[d.provisional] = DeclID(d.kind, d.qualifiedName, a.filePath, d.signatureKey)
		} else {
			final[d.provisional] = OrdinalID(d.kind, d.qualifiedName, a.filePath, d.signatureKey, n)
		}
	}

	result.RewriteIDs(func(id string) string {
		if f, ok := final[id]; ok {
			return f
		}
		return id
	})

	if path, ok := findString(reflect.ValueOf(result), "result", a.seen); ok {
		return fmt.Errorf("nodeid: %s still holds a provisional id after Finalize", path)
	}
	return nil
}

// findString walks the exported fields, elements, map keys and map values
// reachable from v and reports the path of the first string in want.
func findString(v reflect.Value, path string, want map[string]bool) (string, bool) {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "", false
		}
		return findString(v.Elem(), path, want)
	case reflect.String:
		return path, want[v.String()]
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if p, ok := findString(v.Field(i), path+"."+t.Field(i).Name, want); ok {
				return p, true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if p, ok := findString(v.Index(i), fmt.Sprintf("%s[%d]", path, i), want); ok {
				return p, true
			}
		}
	case reflect.Map:
		it := v.MapRange()
		for it.Next() {
			if p, ok := findString(it.Key(), path+" key", want); ok {
				return p, true
			}
			if p, ok := findString(it.Value(), fmt.Sprintf("%s[%v]", path, it.Key()), want); ok {
				return p, true
			}
		}
	}
	return "", false
}
