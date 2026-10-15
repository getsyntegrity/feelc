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
	"github.com/getsyntegrity/feelc/internal/project"
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

// ProjectModule is one module of a project, in memory.
type ProjectModule struct {
	// Name is the module's logical identity: [A-Za-z][A-Za-z0-9_]*, at most 64 characters, unique in the
	// project. It must not contain "__", which is reserved as the namespace separator; such a name is rejected.
	Name string
	// Source is the module's rules source text.
	Source string
	// Uses binds a declared input of this module (local alias) to another module's decision, written "module.decision".
	Uses map[string]string
}

// CompileProject compiles and links the modules of a project, in memory, into one merged model.
// name labels the project; mods are the modules in any order.
//
// The returned string is the merged project's identity hash (sha256 of the merged model's IR
// encoding). For a single module it equals the ID that Compile returns for the same source. It
// carries the same limitations as Compile's ID: it identifies the compiled model, not the source
// text (two sources that compile to the same IR share it), and it says nothing about the model's
// verification status.
//
// CompileProject does not enable publication or activation; it only compiles and links. Each
// module is verified by the linker but verification blockers are NOT returned as errors, so the
// consumer must call Verify on the merged model and block activation when it reports blocking
// findings.
//
// On any error it returns nil, "", err. The caller's Uses maps are copied and never shared.
func CompileProject(name string, mods []ProjectModule) (*Model, string, error) {
	conv := make([]project.SourceModule, len(mods))
	for i, m := range mods {
		var uses map[string]string
		if m.Uses != nil {
			uses = make(map[string]string, len(m.Uses))
			for k, v := range m.Uses {
				uses[k] = v
			}
		}
		conv[i] = project.SourceModule{Name: m.Name, Source: m.Source, Uses: uses}
	}
	p, err := project.Compile(name, conv)
	if err != nil {
		return nil, "", err
	}
	return p.Merged, p.Hash, nil
}

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
