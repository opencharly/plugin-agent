package agentkind

// schema_served_test.go is the schema-LOAD gate, and the §10.1 S3 gate.
//
// `charly box validate` does NOT compile a served schema: a dangling reference
// inside a served def is green at plugin load and only surfaces when the host
// validates an authored `agent:` / `agent-team:` body against it
// (opencharly/spec#301). So this file composes the plugin's OWN Describe reply and
// compiles the body it serves, with the same CUE library the runtime links — the
// served body, not the file on disk, is the surface the host splices onto its base.
//
// It pins the S3 half of D-PAT-3 as well: every closed set of words is a vocabulary
// table, its word list is DERIVED from that table, and the enum an author meets is
// `or()` of the derived list — so a word cannot be added to the table and forgotten
// by the list, nor accepted by the enum while the table it claims to derive from
// lacks it.

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	pb "github.com/opencharly/spec/proto"
)

// vocabHarness LIFTS the hidden vocabulary tables and their derived word lists into
// one public definition. An `_`-prefixed label is hidden by CUE and cannot be looked
// up from outside the compilation unit, so the served body alone exposes only the
// enums; this fragment adds no semantics, it re-exports the tables the schema already
// carries so the test can read them instead of restating their words.
//
// The lists are read through `words.<name>` — a REQUIRED field of this definition —
// rather than at the defs' own fields, four of which are optional (and a plain
// LookupPath refuses an optional field).
const vocabHarness = `
#TestVocabulary: {
	tables: {
		mode:            _mode
		evidence_policy: _evidencePolicy
		transport:       _transport
		persistence:     _persistence
		transcript:      _transcript
	}
	words: {
		mode:            _modeWord
		evidence_policy: _evidencePolicyWord
		transport:       _transportWord
		persistence:     _persistenceWord
		transcript:      _transcriptWord
	}
}
`

// vocabProbe pairs a vocabulary table (named as in vocabHarness) with ONE authored
// document that uses a word from it, so the enum is exercised through the same def the
// host validates against.
type vocabProbe struct {
	name  string // the key vocabHarness gives this table
	label string // the table's own label in the served body
	use   func(word string) string
}

// vocabProbes covers every vocabulary table this schema carries, and every field wired
// to it. A new table belongs in vocabHarness and here.
var vocabProbes = []vocabProbe{
	{"mode", "_mode", func(w string) string {
		return fmt.Sprintf(`{#AgentInput: {command: ["c"], credential: [{src: "/h", dst: "/g", mode: %q}]}}`, w)
	}},
	{"evidence_policy", "_evidencePolicy", func(w string) string {
		return fmt.Sprintf(`{#AgentTeamInput: {agents: [{name: "a", runtime: "pi"}], evidence_policy: %q}}`, w)
	}},
	{"transport", "_transport", func(w string) string {
		return fmt.Sprintf(`{#AgentTeamInput: {agents: [{name: "a", runtime: "pi", target: {hops: [{transport: %q}]}}]}}`, w)
	}},
	{"persistence", "_persistence", func(w string) string {
		return fmt.Sprintf(`{#AgentTeamInput: {agents: [{name: "a", runtime: "pi", terminal_profile: {name: "t", entrypoint: ["bash"], persistence: %q}}]}}`, w)
	}},
	{"transcript", "_transcript", func(w string) string {
		return fmt.Sprintf(`{#AgentTeamInput: {agents: [{name: "a", runtime: "pi", terminal_profile: {name: "t", entrypoint: ["bash"], transcript: %q}}]}}`, w)
	}},
}

// servedSchema returns the plugin's served CUE body COMPILED, with the test's table
// re-export appended. NewMeta's Describe already compiles the concatenated body
// (sdk.BuildCapabilities), so a failure here means the schema the host receives does
// not stand on its own.
func servedSchema(t *testing.T) cue.Value {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	body := caps.GetSchemaCue()
	if body == "" {
		t.Fatal("Describe served an EMPTY CUE body")
	}
	v := cuecontext.New().CompileString(body + vocabHarness)
	if err := v.Err(); err != nil {
		t.Fatalf("served schema does not compile: %v", err)
	}
	return v
}

// schemaWordList returns the sorted alternatives of a disjunction at path — for an S3
// enum, exactly the words its derived list feeds.
//
// The alternatives are read from the EXPRESSION, not from the text and not through
// `String()`: on a disjunction that carries a default `String()` hands back the
// default, which would read a whole catalog as one word.
func schemaWordList(t *testing.T, v cue.Value, path string) []string {
	t.Helper()
	seen := map[string]bool{}
	collectWords(t, path, v.LookupPath(cue.ParsePath(path)), 0, seen)
	words := make([]string, 0, len(seen))
	for w := range seen {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func collectWords(t *testing.T, path string, val cue.Value, depth int, seen map[string]bool) {
	t.Helper()
	if err := val.Err(); err != nil {
		t.Fatalf("%s does not resolve in the served schema: %v", path, err)
	}
	if depth > 16 {
		t.Fatalf("%s: the disjunction nests deeper than 16 levels", path)
	}
	if val.IsConcrete() && val.IncompleteKind() == cue.StringKind {
		w, err := val.String()
		if err != nil {
			t.Fatalf("%s: a concrete string does not stringify: %v", path, err)
		}
		seen[w] = true
		return
	}
	_, alts := val.Expr()
	if len(alts) == 0 {
		t.Fatalf("%s has no alternatives — it is not a word list", path)
	}
	for _, alt := range alts {
		collectWords(t, path, alt, depth+1, seen)
	}
}

// TestServedSchemaResolvesItsDefs — every definition the plugin's contract names must
// still resolve in the served body.
func TestServedSchemaResolvesItsDefs(t *testing.T) {
	v := servedSchema(t)
	for _, path := range []string{
		"#AgentInput", "#AgentTeamInput", "#AgDuration", "#AgCredentialMount",
		"#AtMember", "#AtEdge", "#AtTarget", "#AtTargetHop", "#AtTerminalProfile",
	} {
		if err := v.LookupPath(cue.ParsePath(path)).Err(); err != nil {
			t.Errorf("served schema: %s does not resolve: %v", path, err)
		}
	}
}

// TestVocabularyTablesNameTheirOwner pins the §3.1 ownership tag: each table carries
// `@vocab(<the plugin's own word>)`, so a table copied out of another plugin's schema
// cannot pass as this one's.
//
// The assertion is made on the served TEXT because a `@vocab` is a compile-time
// annotation: it does not survive a reference, so it cannot be read through the
// harness's `tables.<name>` lift. It is the attribute and ONLY the attribute that is
// read as text here — no word set is ever counted from the text.
func TestVocabularyTablesNameTheirOwner(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	body := caps.GetSchemaCue()
	tags := regexp.MustCompile(`@vocab\(([^)]*)\)`).FindAllStringSubmatch(body, -1)
	if len(tags) != len(vocabProbes) {
		t.Fatalf("served body carries %d @vocab tags for %d vocabulary tables — every table is tagged exactly once", len(tags), len(vocabProbes))
	}
	for _, tag := range tags {
		if tag[1] != "agent" {
			t.Errorf("a vocabulary table is tagged @vocab(%s), want @vocab(agent) — the table claims an owner that is not this plugin", tag[1])
		}
	}
}

// TestS3VocabularyTablesDriveTheirEnums proves, for every vocabulary table, that (a)
// the derived word list IS the table's row set, and (b) each word is accepted where an
// author uses it while a non-word is refused. (b) is what a table that is merely
// DECLARED — never wired into the enum — fails.
func TestS3VocabularyTablesDriveTheirEnums(t *testing.T) {
	v := servedSchema(t)
	ctx := v.Context()
	var lifted struct {
		Tables map[string]map[string]any
		Words  map[string][]string
	}
	if err := v.LookupPath(cue.ParsePath("#TestVocabulary")).Decode(&lifted); err != nil {
		t.Fatalf("lift the vocabulary tables: %v", err)
	}
	for _, p := range vocabProbes {
		rows, ok := lifted.Tables[p.name]
		if !ok {
			t.Errorf("no vocabulary table %q in the served schema", p.name)
			continue
		}
		if len(rows) == 0 {
			t.Errorf("vocabulary table %q has no rows", p.name)
			continue
		}
		words, ok := lifted.Words[p.name]
		if !ok {
			t.Errorf("no derived word list for %q in the served schema", p.name)
			continue
		}
		keys := make([]string, 0, len(rows))
		for w := range rows {
			keys = append(keys, w)
		}
		sort.Strings(keys)
		sort.Strings(words)
		if fmt.Sprint(keys) != fmt.Sprint(words) {
			t.Errorf("%q's derived list %v is not its table's row set %v", p.name, words, keys)
		}
		for _, w := range keys {
			if err := v.Unify(ctx.CompileString(p.use(w))).Err(); err != nil {
				t.Errorf("%q word %q is refused where an author uses it: %v", p.name, w, err)
			}
		}
		if err := v.Unify(ctx.CompileString(p.use("zzz-not-a-word"))).Err(); err == nil {
			t.Errorf("%q accepts a non-word — the enum is not the table", p.name)
		}
	}
}

// appendTableRow returns the served body with one word APPENDED to the named table's
// declaration. The tables are written on one line each — `_name: {a: {}, b: {}} @vocab(x)`
// — so the row can be inserted without parsing CUE. A table that stops being one line
// fails this LOUDLY rather than silently mutating the wrong text.
func appendTableRow(t *testing.T, body, table, word string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(table) + `): \{(.*)\} @vocab\(`)
	loc := re.FindStringSubmatchIndex(body)
	if loc == nil {
		t.Fatalf("could not locate the one-line declaration of %s in the served body — this probe edits the TABLE, so a reformatted table must be taught to it", table)
	}
	rows := body[loc[4]:loc[5]]
	return body[:loc[4]] + rows + `, "` + word + `": {}` + body[loc[5]:]
}

// TestATableWordReachesItsField proves the DERIVATION and not merely the equality of
// two sets: a word appended to a table must appear in the enum the field accepts.
//
// This is the direction a set comparison cannot see. A field written as a literal list
// whose words happen to equal its table passes every equality check — and it is exactly
// the form that drifts the moment the table gains a row, because the literal was never
// wired to anything. Only a MUTATED table distinguishes the two, so the probe mutates.
func TestATableWordReachesItsField(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	body := caps.GetSchemaCue()
	ctx := cuecontext.New()
	const extra = "zzz-gained-word" // quoted in the row, so a hyphen is legal
	for _, p := range vocabProbes {
		v := ctx.CompileString(appendTableRow(t, body, p.label, extra))
		if err := v.Err(); err != nil {
			t.Fatalf("%s: the body with a row added to %s does not compile: %v", p.name, p.label, err)
		}
		if err := v.Unify(ctx.CompileString(p.use(extra))).Err(); err != nil {
			t.Errorf("the field carrying %q does not accept %q after %s gained that row — the field is NOT derived from the table, it only happens to agree with it today: %v", p.name, extra, p.label, err)
		}
	}
}
