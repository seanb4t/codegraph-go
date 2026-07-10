// Package wazero implements the Option-B spike arm of the parser.Parser
// interface (D-05) — a tree-sitter core + grammar set compiled to a single
// self-contained WASM module, run entirely inside a wazero sandbox instead
// of via CGo.
//
// SPIKE ARTIFACT (Plan 01-07 / decision D-05): this package exists solely to
// produce real, measured throughput / incremental-reparse / crash-isolation
// numbers against the CGo arm (internal/parser/cgo) for
// PARSER-DECISION.md. It is NOT proposed for production use as-is and is
// expected to be deleted if D-05a selects Option A (CGo).
//
// WASM artifact provenance: no mature, ready-made wazero distribution of
// tree-sitter exists today (RESEARCH §"State of the Art"; the official
// maintainers explicitly declined a wazero backend in
// tree-sitter/go-tree-sitter#16), and this execution environment has
// neither zig nor wasi-sdk installed to build a grammar-to-WASM pipeline
// from source (RESEARCH §"Environment Availability"). To obtain a REAL
// wazero benchmark within the spike's bounded budget, this package embeds a
// prebuilt, statically-linked WASM binary from github.com/malivvan/tree-sitter
// (MIT, pre-release/experimental — the closest prior art RESEARCH already
// names). See assets/PROVENANCE.md for the exact source commit, license
// text, and the ABI limitations this vendored binary carries. It is vendored
// as data run inside wazero's sandbox, not as a Go module dependency, and is
// never shipped in the production binary regardless of which arm D-05a
// selects.
//
// ABI and known limitations: the vendored binary exports a narrow C ABI
// (ts_parser_new, ts_parser_set_language, ts_parser_parse_string,
// ts_parser_delete, malloc/free, tree_sitter_<lang> language constructors,
// and a handful of ts_node_* accessors) sufficient for full-parse
// benchmarking. It does NOT export ts_tree_edit, ts_tree_get_changed_ranges,
// or ts_tree_delete:
//
//   - Without ts_tree_edit, this arm cannot perform a true
//     InputEdit-annotated incremental reparse (RESEARCH §"Code Examples").
//     Parse still accepts a previous Tree as a hint (passed as
//     ts_parser_parse_string's old_tree argument) when the hint came from
//     the same parser instance, but this is weaker than a properly-edited
//     incremental reparse — see PARSER-DECISION.md for how the benchmark
//     handles this and what it means for the decision.
//   - Without ts_tree_delete, parsed trees cannot be explicitly freed from
//     the WASM guest's heap, so repeated parses accumulate guest memory
//     across a benchmark run. Callers driving many Parse calls against one
//     Parser should budget accordingly (see assets/PROVENANCE.md).
package wazero

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"sync"

	wzr "github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/seanb4t/codegraph-go/internal/parser"
)

//go:embed assets/ts.wasm.gz
var embeddedWasmGZ []byte

// requiredExports are the ABI functions every Parser instance needs,
// regardless of language. See assets/PROVENANCE.md for the full export
// table this vendored binary provides and what it deliberately omits.
var requiredExports = []string{
	"malloc",
	"free",
	"strlen",
	"ts_parser_new",
	"ts_parser_delete",
	"ts_parser_set_language",
	"ts_parser_parse_string",
	"ts_language_version",
	"ts_tree_root_node",
	"ts_node_is_error",
}

var (
	initOnce     sync.Once
	initErr      error
	sharedRT     wzr.Runtime
	sharedModule wzr.CompiledModule
)

// initRuntime compiles the embedded WASM binary exactly once and caches the
// wazero Runtime + CompiledModule for reuse across every Parser instance
// (each instance gets its own instantiated module / linear memory below;
// only the compiled bytecode is shared).
func initRuntime(ctx context.Context) error {
	initOnce.Do(func() {
		raw, err := decompress(embeddedWasmGZ)
		if err != nil {
			initErr = fmt.Errorf("wazero parser: decompress embedded ts.wasm: %w", err)
			return
		}

		sharedRT = wzr.NewRuntime(ctx)
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, sharedRT); err != nil {
			initErr = fmt.Errorf("wazero parser: instantiate WASI snapshot preview1: %w", err)
			return
		}

		sharedModule, err = sharedRT.CompileModule(ctx, raw)
		if err != nil {
			initErr = fmt.Errorf("wazero parser: compile embedded ts.wasm: %w", err)
			return
		}
	})
	return initErr
}

func decompress(gz []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("open gzip reader: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decompressed bytes: %w", err)
	}
	return out, nil
}

// wasmTree wraps the WASM guest-memory pointer to a parsed TSTree, plus the
// owning module instance. A tree pointer is only a valid offset within the
// linear memory of the module instance that produced it — Parse defends
// against a caller passing a Tree produced by a different WParser instance
// by refusing to reuse it (falling back to a fresh parse) rather than
// risking a guest memory access with a foreign offset.
type wasmTree struct {
	ptr   uint64
	owner api.Module
}

// WParser implements parser.Parser using the embedded WASM tree-sitter
// build, running one dedicated module instance (and therefore one isolated
// linear memory) per Parser value.
type WParser struct {
	mu     sync.Mutex
	mod    api.Module
	fns    map[string]api.Function
	parser uint64
	lang   uint64
	closed bool
}

// NewGoParser returns a WParser configured for the Go grammar.
func NewGoParser() (*WParser, error) {
	return newWParser(context.Background(), "tree_sitter_go")
}

// NewPythonParser returns a WParser configured for the Python grammar.
// Python exercises the same external-scanner-derived INDENT/DEDENT
// complexity as the CGo arm's Python parser, making it the spike's
// crash-isolation partner language (D-05).
func NewPythonParser() (*WParser, error) {
	return newWParser(context.Background(), "tree_sitter_python")
}

func newWParser(ctx context.Context, langExport string) (*WParser, error) {
	if err := initRuntime(ctx); err != nil {
		return nil, err
	}

	cfg := wzr.NewModuleConfig().
		WithName("").
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime().
		WithRandSource(rand.Reader)

	mod, err := sharedRT.InstantiateModule(ctx, sharedModule, cfg)
	if err != nil {
		return nil, fmt.Errorf("wazero parser: instantiate module: %w", err)
	}

	fns := make(map[string]api.Function, len(requiredExports))
	for _, name := range requiredExports {
		fn := mod.ExportedFunction(name)
		if fn == nil {
			_ = mod.Close(ctx)
			return nil, fmt.Errorf("wazero parser: embedded ts.wasm missing required export %q", name)
		}
		fns[name] = fn
	}

	langFn := mod.ExportedFunction(langExport)
	if langFn == nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wazero parser: embedded ts.wasm missing language export %q", langExport)
	}

	langRes, err := langFn.Call(ctx)
	if err != nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wazero parser: calling %s: %w", langExport, err)
	}

	parserRes, err := fns["ts_parser_new"].Call(ctx)
	if err != nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wazero parser: ts_parser_new: %w", err)
	}

	okRes, err := fns["ts_parser_set_language"].Call(ctx, parserRes[0], langRes[0])
	if err != nil {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wazero parser: ts_parser_set_language: %w", err)
	}
	if okRes[0] == 0 {
		_ = mod.Close(ctx)
		return nil, fmt.Errorf("wazero parser: ts_parser_set_language rejected grammar %q (version mismatch)", langExport)
	}

	return &WParser{
		mod:    mod,
		fns:    fns,
		parser: parserRes[0],
		lang:   langRes[0],
	}, nil
}

// Parse enforces the shared size ceiling (parser.MaxSourceBytes) before
// handing bytes to the WASM guest (Security Domain V5 / threat T-01-03),
// then calls the embedded ts_parser_parse_string export. If oldTree was
// produced by this exact WParser instance, its guest pointer is passed as
// the old_tree hint; a Tree from any other instance (or backend) is safely
// ignored rather than risking a foreign-memory read (see the ABI
// limitations note in the package doc — this is not a true
// InputEdit-annotated incremental reparse).
func (p *WParser) Parse(source []byte, oldTree *parser.Tree) (*parser.Tree, error) {
	if len(source) > parser.MaxSourceBytes {
		return nil, parser.ErrSourceTooLarge
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("wazero parser: Parse called after Close")
	}

	ctx := context.Background()

	var oldTreePtr uint64
	if oldTree != nil {
		if wt, ok := oldTree.Inner().(*wasmTree); ok && wt.owner == p.mod {
			oldTreePtr = wt.ptr
		}
	}

	var strPtr uint64
	if len(source) > 0 {
		mallocRes, err := p.fns["malloc"].Call(ctx, uint64(len(source)))
		if err != nil {
			return nil, fmt.Errorf("wazero parser: malloc: %w", err)
		}
		strPtr = mallocRes[0]
		defer func() {
			_, _ = p.fns["free"].Call(ctx, strPtr)
		}()

		if !p.mod.Memory().Write(uint32(strPtr), source) {
			return nil, errors.New("wazero parser: writing source bytes to guest memory failed")
		}
	}

	treeRes, err := p.fns["ts_parser_parse_string"].Call(ctx, p.parser, oldTreePtr, strPtr, uint64(len(source)))
	if err != nil {
		// A WASM trap (e.g. an out-of-bounds guest access or exhausted
		// linear memory) surfaces here as a plain Go error rather than
		// crashing this process — the isolation property the spike's
		// crash-isolation dimension measures against the CGo arm's
		// unrecoverable C-level segfault risk (see parser.Parser's
		// crash-isolation contract doc and PARSER-DECISION.md).
		return nil, fmt.Errorf("wazero parser: ts_parser_parse_string trapped: %w", err)
	}
	if treeRes[0] == 0 {
		return nil, errors.New("wazero parser: ts_parser_parse_string returned a null tree")
	}

	return parser.NewTree(&wasmTree{ptr: treeRes[0], owner: p.mod}), nil
}

// Close releases the WASM module instance backing this Parser (its entire
// linear memory, including every tree it ever parsed — see the package doc
// for why parsed trees cannot be freed individually). Safe to call once;
// repeat calls are a no-op.
func (p *WParser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}
	p.closed = true

	ctx := context.Background()
	if _, err := p.fns["ts_parser_delete"].Call(ctx, p.parser); err != nil {
		_ = p.mod.Close(ctx)
		return fmt.Errorf("wazero parser: ts_parser_delete: %w", err)
	}
	if err := p.mod.Close(ctx); err != nil {
		return fmt.Errorf("wazero parser: closing module instance: %w", err)
	}
	return nil
}

var _ parser.Parser = (*WParser)(nil)
