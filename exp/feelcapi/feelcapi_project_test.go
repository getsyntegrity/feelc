package feelcapi_test

import (
	"strings"
	"testing"

	"github.com/getsyntegrity/feelc/exp/feelcapi"
)

const srcB = `model "b" {
  rounding: half_even
}

input score : number >= 0

decision passed : boolean = score >= 600
`

const srcA = `model "a" {
  rounding: half_even
}

input amount : number >= 0
input x : boolean

decision approved : boolean {
  needs: x, amount
  hit: first
  #  x     | amount      => approved
     false | -           => false
     -     | > 100000    => false
     true  | <= 100000   => true
     default|            => false
}
`

// srcUniqueConflict links fine but its UNIQUE table overlaps, which the verifier reports as a blocker.
const srcUniqueConflict = `model "u" {}
input n : number in [0..100]
decision d : string {
  needs: n
  hit: unique
     >= 0  => "a"
     >= 50 => "b"
}
`

func linkedProject(t *testing.T) (*feelcapi.Model, string) {
	t.Helper()
	m, h, err := feelcapi.CompileProject("lending", []feelcapi.ProjectModule{
		{Name: "b", Source: srcB},
		{Name: "a", Source: srcA, Uses: map[string]string{"x": "b.passed"}},
	})
	if err != nil {
		t.Fatalf("CompileProject: %v", err)
	}
	return m, h
}

func requireFailure(t *testing.T, m *feelcapi.Model, h string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	if m != nil || h != "" {
		t.Fatalf("on error want nil model and empty hash, got model=%v hash=%q", m != nil, h)
	}
}

func TestCompileProjectLinksModules(t *testing.T) {
	m, h := linkedProject(t)
	if h == "" {
		t.Fatal("empty hash")
	}
	got, err := feelcapi.Evaluate(m, "a__approved", map[string]any{"b__score": 700, "a__amount": 50000})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got != true {
		t.Errorf("approved(score=700) = %v, want true (value flows from b.passed)", got)
	}
	got, _ = feelcapi.Evaluate(m, "a__approved", map[string]any{"b__score": 500, "a__amount": 50000})
	if got != false {
		t.Errorf("approved(score=500) = %v, want false", got)
	}
	if n := feelcapi.Verify(m).Blockers(); n != 0 {
		t.Errorf("Verify blockers = %d, want 0", n)
	}
}

func TestCompileProjectDuplicateName(t *testing.T) {
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "b", Source: srcB}, {Name: "b", Source: srcB},
	})
	requireFailure(t, m, h, err)
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("error %q should report the duplicate", err)
	}
}

func TestCompileProjectUnknownModule(t *testing.T) {
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "b", Source: srcB},
		{Name: "a", Source: srcA, Uses: map[string]string{"x": "nope.passed"}},
	})
	requireFailure(t, m, h, err)
	for _, want := range []string{"a", "x", "nope"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestCompileProjectUnknownDecision(t *testing.T) {
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "b", Source: srcB},
		{Name: "a", Source: srcA, Uses: map[string]string{"x": "b.ghost"}},
	})
	requireFailure(t, m, h, err)
	for _, want := range []string{"a", "x", "ghost"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestCompileProjectCycle(t *testing.T) {
	a := "model \"a\" {}\ninput from_b : boolean\ndecision a_out : boolean = from_b\n"
	b := "model \"b\" {}\ninput from_a : boolean\ndecision b_out : boolean = from_a\n"
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "a", Source: a, Uses: map[string]string{"from_b": "b.b_out"}},
		{Name: "b", Source: b, Uses: map[string]string{"from_a": "a.a_out"}},
	})
	requireFailure(t, m, h, err)
	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("error %q should report the cycle", err)
	}
}

func TestCompileProjectSyntaxErrorHasLocation(t *testing.T) {
	bad := "model \"b\" {}\ninput score : number\ndecision passed : boolean = score >=\n"
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "a", Source: srcA, Uses: nil},
		{Name: "b", Source: bad},
	})
	requireFailure(t, m, h, err)
	if !strings.Contains(err.Error(), "b.rules:") {
		t.Errorf("error %q should name b.rules", err)
	}
	if !strings.Contains(err.Error(), "b.rules:3:") {
		t.Errorf("error %q should point at line 3", err)
	}
}

func TestCompileProjectInvalidModuleName(t *testing.T) {
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{{Name: "1bad", Source: srcB}})
	requireFailure(t, m, h, err)
}

// Compile/link and verification are separate: the project links (err == nil) while the model is
// still blocked at verification, so the caller must run Verify before activating.
func TestCompileProjectLinksButVerifyBlocks(t *testing.T) {
	m, h, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{{Name: "u", Source: srcUniqueConflict}})
	if err != nil || m == nil || h == "" {
		t.Fatalf("project must link: model=%v hash=%q err=%v", m != nil, h, err)
	}
	if n := feelcapi.Verify(m).Blockers(); n == 0 {
		t.Error("Verify should report a blocking UNIQUE conflict")
	}
}

func TestCompileProjectDoesNotAliasUses(t *testing.T) {
	uses := map[string]string{"x": "b.passed"}
	m, _, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{
		{Name: "b", Source: srcB}, {Name: "a", Source: srcA, Uses: uses},
	})
	if err != nil {
		t.Fatal(err)
	}
	uses["x"] = "b.ghost"
	uses["extra"] = "b.ghost"
	got, err := feelcapi.Evaluate(m, "a__approved", map[string]any{"b__score": 700, "a__amount": 1})
	if err != nil || got != true {
		t.Errorf("result changed after caller mutated Uses: %v, %v", got, err)
	}
}

func TestCompileProjectSingleModuleHashEqualsCompile(t *testing.T) {
	_, want, _, err := feelcapi.Compile([]byte(srcB))
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := feelcapi.CompileProject("p", []feelcapi.ProjectModule{{Name: "b", Source: srcB}})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("hash = %s, want %s", got, want)
	}
}
