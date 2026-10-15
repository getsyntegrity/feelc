// Package feelcapi is an EXPERIMENTAL facade over the feelc engine, added by the Verimand fork.
//
// It is not a public API and carries no compatibility promise: it re-exports feelc internal
// types as aliases, so any upstream change to those types changes this package. Only the
// Verimand Rules adapter (#379) may import it. The definitive public API is the adapter's own
// generic contract, which uses its own types and never exposes these aliases.
//
// probe.sh copies this file into exp/feelcapi/feelcapi.go of the probe fork.
package feelcapi

import (
	"github.com/getsyntegrity/feelc/internal/engine"
	"github.com/getsyntegrity/feelc/internal/explain"
	"github.com/getsyntegrity/feelc/internal/ir"
	"github.com/getsyntegrity/feelc/internal/loader"
	"github.com/getsyntegrity/feelc/internal/verify"
)

type (
	Model  = ir.CompiledModel
	Report = verify.Report
	Trace  = explain.Trace
)

// Compile parses, type-checks, lowers and verifies a .rules source and returns the model
// identity hash (sha256 of its IR encoding).
func Compile(src []byte) (*Model, string, *Report, error) { return loader.Compile(src) }

// Verify re-runs the static verifier on a model.
func Verify(m *Model) *Report { return verify.Verify(m) }

// Evaluate runs one decision of a compiled model.
func Evaluate(m *Model, decision string, in map[string]any) (any, error) {
	return engine.Eval(m, decision, in)
}

// Explain returns the justification trace for one decision.
func Explain(m *Model, decision string, in map[string]any) (*Trace, error) {
	return explain.Explain(m, decision, in)
}

// Encode and Decode expose the .ir.bin codec.
func Encode(m *Model) ([]byte, error) { return ir.Encode(m) }
func Decode(b []byte) (*Model, error) { return ir.Decode(b) }
