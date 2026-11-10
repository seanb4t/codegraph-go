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

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(&rec); err != nil {
		return nil, fmt.Errorf("spill: encode %s: %w", r.RelPath, err)
	}
	return buf.Bytes(), nil
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
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&rec); err != nil {
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
