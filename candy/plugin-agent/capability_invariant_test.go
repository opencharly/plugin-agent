package agentkind

// capability_invariant_test.go pins the catalog invariant THIS plugin's shape admits.
//
// The verb plugins in this sweep (jetkvm, cua, punktfunk, gh) pair a served enum with a
// dispatch switch: every advertised word must be routed and every routed arm must be
// advertised. This plugin is a KIND plugin and has no op switch — its catalog is the
// CAPABILITY list Describe advertises against a set of input defs. So the same two
// directions read, for this shape:
//
//   - every input def a capability advertises must RESOLVE in the served body, or the
//     host's splice produces a def it can never look up and the first authored body of
//     that kind fails at registration-time rather than at authoring time;
//   - every `#…Input` def the served body carries must be ADVERTISED by a capability,
//     or it is dead surface: a def an author is invited to fill and no word dispatches.
//
// Both readers read the REAL sources — the served body through the CUE API
// (schema_served_test.go) and Describe's own provided list — never a restatement.

import (
	"context"
	"regexp"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	pb "github.com/opencharly/spec/proto"
)

// advertisedInputDefs returns the input defs Describe advertises, sorted.
func advertisedInputDefs(t *testing.T) []string {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	seen := map[string]bool{}
	for _, provided := range caps.GetProvided() {
		if def := provided.GetInputDef(); def != "" {
			seen[def] = true
		}
	}
	if len(seen) == 0 {
		t.Fatal("Describe advertises no input def — the invariant would pass over nothing")
	}
	return sortedKeys(seen)
}

// servedInputDefs returns every top-level `#…Input` definition the SERVED body carries,
// read from the served text's declaration labels. The DEFINITION NAMES are read here,
// not a word set: an enum has no textual form to parse any more, and no word is counted
// from this text.
func servedInputDefs(t *testing.T) []string {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`(?m)^#([A-Za-z0-9_]*Input):`).FindAllStringSubmatch(caps.GetSchemaCue(), -1) {
		seen["#"+m[1]] = true
	}
	if len(seen) == 0 {
		t.Fatal("served body declares no #…Input definition — the invariant would pass over nothing")
	}
	return sortedKeys(seen)
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestEveryAdvertisedInputDefResolves — a capability naming a def the served body does
// not carry is a registration-time surprise for the host.
func TestEveryAdvertisedInputDefResolves(t *testing.T) {
	v := servedSchema(t)
	for _, def := range advertisedInputDefs(t) {
		if err := v.LookupPath(cue.ParsePath(def)).Err(); err != nil {
			t.Errorf("capability advertises %s, which does not resolve in the served body: %v", def, err)
		}
	}
}

// TestNoUnadvertisedInputDef is the other direction: a served `#…Input` def no
// capability advertises is dead surface.
func TestNoUnadvertisedInputDef(t *testing.T) {
	advertised := map[string]bool{}
	for _, def := range advertisedInputDefs(t) {
		advertised[def] = true
	}
	for _, def := range servedInputDefs(t) {
		if !advertised[def] {
			t.Errorf("served body declares %s, which NO capability advertises — an author is invited to fill a body no word dispatches", def)
		}
	}
}

// TestEveryAdvertisedInputDefIsAccepted is the two-way set equality in one assertion, so
// a reader sees the whole invariant at once.
func TestEveryAdvertisedInputDefIsAccepted(t *testing.T) {
	advertised := advertisedInputDefs(t)
	served := servedInputDefs(t)
	if len(advertised) != len(served) {
		t.Fatalf("Describe advertises %d input defs but the served body declares %d — the catalog and the schema have drifted", len(advertised), len(served))
	}
	for i := range advertised {
		if advertised[i] != served[i] {
			t.Errorf("advertised input def %q has no matching declaration in the served body (served=%v)", advertised[i], served)
			break
		}
	}
}
