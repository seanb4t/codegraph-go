package symbols

import (
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// IncludeCandidates returns the paths that the written path p of an
// include in file can name, in the order of the lookup: p beside file,
// then p from the repository root. A path that starts with "/" has none,
// and so has a path with a ".", ".." or empty segment after its leading
// "./" and "../" segments. A candidate outside the root drops out.
func IncludeCandidates(file, p string) []string {
	if strings.HasPrefix(p, "/") {
		return nil
	}
	segs := strings.Split(p, "/")
	i := 0
	for i < len(segs) && (segs[i] == "." || segs[i] == "..") {
		i++
	}
	if i == len(segs) {
		return nil
	}
	for _, s := range segs[i:] {
		if s == "" || s == "." || s == ".." {
			return nil
		}
	}
	var out []string
	for _, c := range []string{path.Join(path.Dir(file), p), path.Clean(p)} {
		if fs.ValidPath(c) && !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	return out
}

// includeMatch returns the held file that the written path p of an
// include in file names, or "". The lookup ends at a candidate equal to
// file, since the compiler opens the includer itself.
func includeMatch(file, p string, held map[string]bool) string {
	for _, c := range IncludeCandidates(file, p) {
		if c == file {
			return ""
		}
		if held[c] {
			return c
		}
	}
	return ""
}

// IncludeWaits returns the paths at which a new file changes the match
// of the written path p of an include in file, when target is its match
// now or "" (design D2): the candidates before target, up to the
// includer.
func IncludeWaits(file, p, target string) []string {
	var out []string
	for _, c := range IncludeCandidates(file, p) {
		if c == file || c == target {
			break
		}
		out = append(out, c)
	}
	return out
}

// AddStoredInclude records e, the stored imports edge of a C or C++
// include between two file nodes, from a file the sync does not reparse.
// The view keeps its target as it is.
func (b *Builder) AddStoredInclude(e *schema.Edge) {
	d := b.write()
	d.storedIncludes = append(d.storedIncludes, e)
}

// freezeIncludes resolves the written includes against the held C and
// C++ files and adds the stored includes. A file is held when the view
// has its file node and its result has no error.
func (d *data) freezeIncludes() {
	held := make(map[string]bool)
	for _, n := range d.nodes {
		if n.Kind == goextract.KindFile && LinksIncludes(n.Language) && !d.failedFiles[n.FilePath] {
			held[n.FilePath] = true
		}
	}
	d.includeTargets = make(map[string]map[string]string)
	d.includes = make(map[string][]string)
	for file, written := range d.writtenIncludes {
		for _, p := range written {
			target := includeMatch(file, p, held)
			if target == "" {
				continue
			}
			if d.includeTargets[file] == nil {
				d.includeTargets[file] = make(map[string]string)
			}
			d.includeTargets[file][p] = target
			d.includes[file] = append(d.includes[file], target)
		}
	}
	for _, e := range d.storedIncludes {
		from, to := d.nodes[e.Source], d.nodes[e.Target]
		if from == nil || to == nil {
			continue
		}
		d.includes[from.FilePath] = append(d.includes[from.FilePath], to.FilePath)
	}
	for file, targets := range d.includes {
		slices.Sort(targets)
		d.includes[file] = slices.Compact(targets)
	}
}

// IncludeTarget returns the file that the written path p of an include
// in file links to, or "". It knows the includes of the files whose
// declarations the Builder received.
func (v View) IncludeTarget(file, p string) string { return v.d.includeTargets[file][p] }

// IncludesOf returns the files that file includes, in path order.
func (v View) IncludesOf(file string) []string { return v.d.includes[file] }
