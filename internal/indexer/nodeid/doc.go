// Package nodeid computes the deterministic content-hashed identifier
// assigned to every extracted symbol (D-02/D-02a). It is a leaf package —
// it imports only the standard library — so both the extractor and the
// resolver can depend on it without an import cycle.
package nodeid
