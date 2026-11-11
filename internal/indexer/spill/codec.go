// Package spill holds the extraction results of a full index on disk
// between the passes. Pass 1 appends each goextract.FileResult to a spill
// file, and the later passes read each one back by its offset. So a result
// is in memory only while a worker holds its file.
package spill

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/recordio"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// record is the gob form of one FileResult. Gob encodes Result as the Go
// struct that it is, so a new field of FileResult travels with no change
// here. Three things do not go through gob and travel beside it. Gob
// cannot handle the internal fields of a protobuf message, so each node
// and each edge travels as deterministic proto bytes. An error value has
// no gob form, so Err travels as its text.
type record struct {
	// Result is the FileResult with Nodes, IntraEdges and Err cleared.
	Result goextract.FileResult
	Nodes  []wireNode
	Edges  []wireEdge

	HasErr bool
	// ErrText is the Error() text, and ErrInternal says whether the error
	// wraps goextract.ErrInternal.
	ErrText     string
	ErrInternal bool
}

// wireNode is one ExtractedNode: its proto bytes, and the struct with the
// node cleared.
type wireNode struct {
	Proto []byte
	Rest  goextract.ExtractedNode
}

// wireEdge is one IntraEdge, in the same form.
type wireEdge struct {
	Proto []byte
	Rest  goextract.IntraEdge
}

// spilledError is the error of a decoded result. It has the text of the
// original, and it still matches goextract.ErrInternal with errors.Is when
// the original did.
type spilledError struct {
	text     string
	internal bool
}

func (e *spilledError) Error() string { return e.text }

func (e *spilledError) Is(target error) bool {
	return e.internal && target == goextract.ErrInternal
}

// Encode returns the spill bytes of r. It does not change r.
func Encode(r *goextract.FileResult) ([]byte, error) {
	rec := record{Result: *r}
	rec.Result.Nodes, rec.Result.IntraEdges, rec.Result.Err = nil, nil, nil

	rec.Nodes = make([]wireNode, len(r.Nodes))
	for i, en := range r.Nodes {
		data, err := marshalOptional(en.Node == nil, en.Node)
		if err != nil {
			return nil, fmt.Errorf("spill: encode node of %s: %w", r.RelPath, err)
		}
		en.Node = nil
		rec.Nodes[i] = wireNode{Proto: data, Rest: en}
	}
	rec.Edges = make([]wireEdge, len(r.IntraEdges))
	for i, ie := range r.IntraEdges {
		data, err := marshalOptional(ie.Edge == nil, ie.Edge)
		if err != nil {
			return nil, fmt.Errorf("spill: encode edge of %s: %w", r.RelPath, err)
		}
		ie.Edge = nil
		rec.Edges[i] = wireEdge{Proto: data, Rest: ie}
	}
	if r.Err != nil {
		rec.HasErr = true
		rec.ErrText = r.Err.Error()
		rec.ErrInternal = errors.Is(r.Err, goextract.ErrInternal)
	}

	return encodeRecord(&rec, r.RelPath)
}

// A gob stream sends the description of each type once, before the first
// value of that type. A record must be readable on its own, at any offset
// and in any order. A stream of its own for each record would repeat the
// type descriptions in every record and compile a new decoder for every
// read, and for a small file that costs far more than the record itself.
//
// So every record is one value message of a gob stream whose type
// descriptions are known in advance. The first Encode of a gob.Encoder
// sends the descriptions of every type that the static type of record can
// reach, in an order that depends only on that type. So all encoders agree
// on them after one value, and a decoder that read the output of any one
// encoder for the zero record (prelude) can read a value message of every
// other.

// streamEncoder is a gob.Encoder that already sent its type descriptions.
type streamEncoder struct {
	buf bytes.Buffer
	enc *gob.Encoder
}

// newStreamEncoder returns an encoder past its type descriptions, with the
// bytes that it wrote for them and for the zero record.
func newStreamEncoder() (*streamEncoder, []byte, error) {
	e := &streamEncoder{}
	e.enc = gob.NewEncoder(&e.buf)
	if err := e.enc.Encode(&record{}); err != nil {
		return nil, nil, err
	}
	prelude := bytes.Clone(e.buf.Bytes())
	e.buf.Reset()
	return e, prelude, nil
}

// streamDecoder is a gob.Decoder that already read the prelude. src
// implements io.ByteReader, so the decoder reads from it directly and
// keeps no bytes of its own between two records.
type streamDecoder struct {
	src bytes.Reader
	dec *gob.Decoder
}

func newStreamDecoder() (*streamDecoder, error) {
	prelude, err := streamPrelude()
	if err != nil {
		return nil, err
	}
	d := &streamDecoder{}
	d.src.Reset(prelude)
	d.dec = gob.NewDecoder(&d.src)
	if err := d.dec.Decode(&record{}); err != nil {
		return nil, err
	}
	return d, nil
}

// streamPrelude returns the type descriptions that every streamDecoder
// reads first.
var streamPrelude = sync.OnceValues(func() ([]byte, error) {
	_, prelude, err := newStreamEncoder()
	return prelude, err
})

// encoders and decoders hold the idle stream codecs. A codec that returned
// an error does not go back, because its stream state is then unknown.
var encoders, decoders sync.Pool

func encodeRecord(rec *record, relPath string) ([]byte, error) {
	e, _ := encoders.Get().(*streamEncoder)
	if e == nil {
		var err error
		if e, _, err = newStreamEncoder(); err != nil {
			return nil, fmt.Errorf("spill: encode %s: %w", relPath, err)
		}
	}
	if err := e.enc.Encode(rec); err != nil {
		return nil, fmt.Errorf("spill: encode %s: %w", relPath, err)
	}
	data := bytes.Clone(e.buf.Bytes())
	e.buf.Reset()
	encoders.Put(e)
	return data, nil
}

func decodeRecord(data []byte, rec *record) error {
	d, _ := decoders.Get().(*streamDecoder)
	if d == nil {
		var err error
		if d, err = newStreamDecoder(); err != nil {
			return err
		}
	}
	d.src.Reset(data)
	if err := d.dec.Decode(rec); err != nil {
		return err
	}
	if d.src.Len() != 0 {
		return fmt.Errorf("%d bytes after the record", d.src.Len())
	}
	decoders.Put(d)
	return nil
}

// marshalOptional returns the proto bytes of msg with one leading byte that
// says whether the message is present, so a nil message stays nil.
func marshalOptional(isNil bool, msg proto.Message) ([]byte, error) {
	if isNil {
		return []byte{0}, nil
	}
	data, err := recordio.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return append([]byte{1}, data...), nil
}

// unmarshalOptional is the reverse of marshalOptional. It reports whether
// the message was present.
func unmarshalOptional(data []byte, msg proto.Message) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("spill: empty proto field")
	}
	if data[0] == 0 {
		return false, nil
	}
	return true, proto.Unmarshal(data[1:], msg)
}

// Decode returns the FileResult that data holds. An empty slice or map of
// the original comes back as nil, because gob does not send zero values.
func Decode(data []byte) (goextract.FileResult, error) {
	var rec record
	if err := decodeRecord(data, &rec); err != nil {
		return goextract.FileResult{}, fmt.Errorf("spill: decode: %w", err)
	}
	r := rec.Result
	if len(rec.Nodes) > 0 {
		r.Nodes = make([]goextract.ExtractedNode, len(rec.Nodes))
	}
	for i, wn := range rec.Nodes {
		en := wn.Rest
		n := &schema.Node{}
		present, err := unmarshalOptional(wn.Proto, n)
		if err != nil {
			return goextract.FileResult{}, fmt.Errorf("spill: decode node of %s: %w", r.RelPath, err)
		}
		if present {
			en.Node = n
		}
		r.Nodes[i] = en
	}
	if len(rec.Edges) > 0 {
		r.IntraEdges = make([]goextract.IntraEdge, len(rec.Edges))
	}
	for i, we := range rec.Edges {
		ie := we.Rest
		e := &schema.Edge{}
		present, err := unmarshalOptional(we.Proto, e)
		if err != nil {
			return goextract.FileResult{}, fmt.Errorf("spill: decode edge of %s: %w", r.RelPath, err)
		}
		if present {
			ie.Edge = e
		}
		r.IntraEdges[i] = ie
	}
	if rec.HasErr {
		r.Err = &spilledError{text: rec.ErrText, internal: rec.ErrInternal}
	}
	return r, nil
}
