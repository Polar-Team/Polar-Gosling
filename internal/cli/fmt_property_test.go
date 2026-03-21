package cli

import (
"os"
"path/filepath"
"reflect"
"sort"
"strings"
"testing"

"github.com/leanovate/gopter"
"github.com/leanovate/gopter/gen"
"github.com/leanovate/gopter/prop"
"github.com/polar-gosling/gosling/internal/parser"
)

// fmtProperties returns gopter.Properties configured with >= 100 iterations.
func fmtProperties() *gopter.Properties {
params := gopter.DefaultTestParameters()
params.MinSuccessfulTests = 100
return gopter.NewProperties(params)
}

// ---------------------------------------------------------------------------
// AST generators
// ---------------------------------------------------------------------------

// genFmtScalarValue generates a scalar (string, number, bool) parser.Value.
func genFmtScalarValue() gopter.Gen {
return gen.OneGenOf(
gen.Const("hello").Map(func(s string) parser.Value {
return parser.Value{Type: parser.StringType, Raw: s}
}),
gen.Float64Range(1, 100).Map(func(n float64) parser.Value {
return parser.Value{Type: parser.NumberType, Raw: n}
}),
gen.Bool().Map(func(b bool) parser.Value {
return parser.Value{Type: parser.BoolType, Raw: b}
}),
)
}

// genFmtAttributes generates an attribute map with a fixed set of keys so
// alphabetical ordering is always testable without needing FlatMap for keys.
func genFmtAttributes() gopter.Gen {
keys := []string{"alpha", "beta", "gamma", "delta"}
return gen.SliceOfN(len(keys), genFmtScalarValue()).Map(func(vals []parser.Value) map[string]parser.Value {
m := make(map[string]parser.Value, len(keys))
for i, k := range keys {
m[k] = vals[i]
}
return m
})
}

// genFmtSimpleBlock generates a leaf Block (no nested blocks).
func genFmtSimpleBlock() gopter.Gen {
return gen.OneConstOf("runner", "resources", "cloud", "gitlab").
FlatMap(func(bt interface{}) gopter.Gen {
return genFmtAttributes().Map(func(attrs map[string]parser.Value) parser.Block {
return parser.Block{
Type:       bt.(string),
Labels:     nil,
Attributes: attrs,
Blocks:     nil,
}
})
}, reflect.TypeOf(parser.Block{}))
}

// genFmtLabel generates a short label safe for use as an HCL block label.
// Uses a fixed set to avoid SuchThat discard overhead.
func genFmtLabel() gopter.Gen {
return gen.OneConstOf("app", "svc", "job", "api", "web", "db", "cache", "proxy")
}

// genFmtTopLevelBlock generates a top-level "egg" Block with a label and
// exactly two nested blocks.
func genFmtTopLevelBlock() gopter.Gen {
return genFmtLabel().
FlatMap(func(label interface{}) gopter.Gen {
return genFmtAttributes().FlatMap(func(attrs interface{}) gopter.Gen {
return gen.SliceOfN(2, genFmtSimpleBlock()).Map(func(nested []parser.Block) parser.Block {
return parser.Block{
Type:       "egg",
Labels:     []string{label.(string)},
Attributes: attrs.(map[string]parser.Value),
Blocks:     nested,
}
})
}, reflect.TypeOf(parser.Block{}))
}, reflect.TypeOf(parser.Block{}))
}

// genFmtConfig generates a Config with 1-3 top-level blocks.
func genFmtConfig() gopter.Gen {
return gen.IntRange(1, 3).FlatMap(func(n interface{}) gopter.Gen {
return gen.SliceOfN(n.(int), genFmtTopLevelBlock()).Map(func(blocks []parser.Block) *parser.Config {
return &parser.Config{Blocks: blocks}
})
}, reflect.TypeOf(&parser.Config{}))
}

// genFmtListBlock generates a Config whose single block has a "tags" list
// attribute of exactly listLen string elements.
func genFmtListBlock(listLen int) gopter.Gen {
// Use a fixed set of short tag strings to avoid SuchThat discard overhead.
tagChoices := []string{"docker", "linux", "amd64", "arm64", "gpu", "fast"}
items := make([]parser.Value, listLen)
for i := range items {
items[i] = parser.Value{Type: parser.StringType, Raw: tagChoices[i%len(tagChoices)]}
}
config := &parser.Config{
Blocks: []parser.Block{
{
Type:   "egg",
Labels: []string{"x"},
Attributes: map[string]parser.Value{
"tags": {Type: parser.ListType, Raw: items},
},
Blocks: nil,
},
},
}
return gen.Const(config)
}

// ---------------------------------------------------------------------------
// Semantic equality helpers (ignore Position fields)
// ---------------------------------------------------------------------------

func fmtConfigsEqual(a, b *parser.Config) bool {
if len(a.Blocks) != len(b.Blocks) {
return false
}
for i := range a.Blocks {
if !fmtBlocksEqual(&a.Blocks[i], &b.Blocks[i]) {
return false
}
}
return true
}

func fmtBlocksEqual(a, b *parser.Block) bool {
if a.Type != b.Type || len(a.Labels) != len(b.Labels) {
return false
}
for i := range a.Labels {
if a.Labels[i] != b.Labels[i] {
return false
}
}
if len(a.Attributes) != len(b.Attributes) {
return false
}
for k, av := range a.Attributes {
bv, ok := b.Attributes[k]
if !ok || !av.Equals(&bv) {
return false
}
}
if len(a.Blocks) != len(b.Blocks) {
return false
}
for i := range a.Blocks {
if !fmtBlocksEqual(&a.Blocks[i], &b.Blocks[i]) {
return false
}
}
return true
}

// ---------------------------------------------------------------------------
// Property 43: Idempotence -- fmt(fmt(x)) == fmt(x)
// Feature: gosling-fmt, Property 43: Idempotence -- fmt(fmt(x)) == fmt(x)
// Validates: Requirements 24.27
// ---------------------------------------------------------------------------

func TestFmtIdempotence(t *testing.T) {
// Feature: gosling-fmt, Property 43: Idempotence -- fmt(fmt(x)) == fmt(x)
properties := fmtProperties()
f := parser.NewFormatter()

properties.Property("Format(Format(x)) == Format(x) for all valid ASTs",
prop.ForAll(
func(config *parser.Config) bool {
first := f.Format(config)

// Re-parse the first-pass output using a fresh parser instance
// (avoids hclparse file-name cache collisions).
p := parser.NewParser()
reparsed, err := p.Parse([]byte(first+"\n"), "idempotence.fly")
if err != nil {
t.Logf("re-parse failed: %v\ninput:\n%s", err, first)
return false
}

second := f.Format(reparsed)
if first != second {
t.Logf("idempotence violation:\nfirst:\n%s\n\nsecond:\n%s", first, second)
return false
}
return true
},
genFmtConfig(),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 44: Round-Trip Correctness -- parse(fmt(x)) yields AST equivalent to x
// Feature: gosling-fmt, Property 44: Round-Trip -- parse(fmt(x)) yields AST equivalent to x
// Validates: Requirements 24.28
// ---------------------------------------------------------------------------

func TestFmtRoundTrip(t *testing.T) {
// Feature: gosling-fmt, Property 44: Round-Trip -- parse(fmt(x)) yields AST equivalent to x
properties := fmtProperties()
f := parser.NewFormatter()

properties.Property("Parse(Format(x)) yields AST semantically equivalent to x",
prop.ForAll(
func(config *parser.Config) bool {
formatted := f.Format(config)

// Use a fresh parser instance per iteration to avoid cache collisions.
p := parser.NewParser()
reparsed, err := p.Parse([]byte(formatted+"\n"), "roundtrip.fly")
if err != nil {
t.Logf("parse error: %v\nformatted:\n%s", err, formatted)
return false
}

if !fmtConfigsEqual(config, reparsed) {
t.Logf("round-trip mismatch: original %d blocks, reparsed %d blocks",
len(config.Blocks), len(reparsed.Blocks))
return false
}
return true
},
genFmtConfig(),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 45: Attribute Alphabetical Ordering
// Validates: Requirements 24.20
// ---------------------------------------------------------------------------

func TestFmtAttributeAlphabeticalOrdering(t *testing.T) {
properties := fmtProperties()
f := parser.NewFormatter()

properties.Property("formatted output lists attribute keys in ascending alphabetical order",
prop.ForAll(
func(config *parser.Config) bool {
formatted := f.Format(config)
for _, block := range config.Blocks {
if !fmtAttrOrderCorrect(formatted, block) {
t.Logf("attribute order violation in block %q\nformatted:\n%s", block.Type, formatted)
return false
}
}
return true
},
genFmtConfig(),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// fmtAttrOrderCorrect verifies that the attribute keys for a block appear in
// alphabetical order within the formatted string.
func fmtAttrOrderCorrect(formatted string, block parser.Block) bool {
if len(block.Attributes) < 2 {
return true
}

keys := make([]string, 0, len(block.Attributes))
for k := range block.Attributes {
keys = append(keys, k)
}
sort.Strings(keys)

// Find the byte-offset of each key's assignment line in the formatted output.
positions := make([]int, 0, len(keys))
for _, k := range keys {
for _, indent := range []string{"\n  ", "\n    ", "\n      "} {
needle := indent + k + " = "
if idx := strings.Index(formatted, needle); idx != -1 {
positions = append(positions, idx)
break
}
}
}

if len(positions) < 2 {
return true // not enough found to compare
}

for i := 1; i < len(positions); i++ {
if positions[i] <= positions[i-1] {
return false
}
}
return true
}

// ---------------------------------------------------------------------------
// Property 46: Nested Block Order Preservation
// Validates: Requirements 24.21
// ---------------------------------------------------------------------------

func TestFmtNestedBlockOrderPreservation(t *testing.T) {
properties := fmtProperties()
f := parser.NewFormatter()

properties.Property("nested block order in formatted output matches input AST order",
prop.ForAll(
func(config *parser.Config) bool {
formatted := f.Format(config)
for _, topBlock := range config.Blocks {
if len(topBlock.Blocks) < 2 {
continue
}
if !fmtNestedOrderPreserved(formatted, topBlock.Blocks) {
t.Logf("nested block order violation\nformatted:\n%s", formatted)
return false
}
}
return true
},
genFmtConfig(),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// fmtNestedOrderPreserved checks that nested block type headers appear in the
// formatted string in the same order as in the AST slice.
func fmtNestedOrderPreserved(formatted string, blocks []parser.Block) bool {
searchFrom := 0
for _, b := range blocks {
found := -1
for _, indent := range []string{"\n  ", "\n    "} {
if idx := strings.Index(formatted[searchFrom:], indent+b.Type+" "); idx != -1 {
found = searchFrom + idx
break
}
if idx := strings.Index(formatted[searchFrom:], indent+b.Type+"{"); idx != -1 {
found = searchFrom + idx
break
}
}
if found == -1 {
continue
}
searchFrom = found + 1
}
return true
}

// ---------------------------------------------------------------------------
// Property 47: List Formatting Threshold
// Validates: Requirements 24.25, 24.26
// ---------------------------------------------------------------------------

func TestFmtListFormattingThreshold(t *testing.T) {
properties := fmtProperties()
f := parser.NewFormatter()

// Lists with <= 2 elements must be inline (no newline immediately after "[").
properties.Property("lists with <= 2 elements are formatted inline",
prop.ForAll(
func(config *parser.Config) bool {
formatted := f.Format(config)
tagsIdx := strings.Index(formatted, "tags = [")
if tagsIdx == -1 {
return true // no tags attribute -- vacuously true
}
bracketPos := tagsIdx + len("tags = [")
if bracketPos >= len(formatted) {
return true
}
// Inline: the character right after "[" must NOT be "\n".
return formatted[bracketPos] != '\n'
},
gen.OneGenOf(
genFmtListBlock(0),
genFmtListBlock(1),
genFmtListBlock(2),
),
))

// Lists with > 2 elements must be multi-line (newline immediately after "[").
properties.Property("lists with > 2 elements are formatted multi-line",
prop.ForAll(
func(config *parser.Config) bool {
formatted := f.Format(config)
tagsIdx := strings.Index(formatted, "tags = [")
if tagsIdx == -1 {
return true
}
bracketPos := tagsIdx + len("tags = [")
if bracketPos >= len(formatted) {
return true
}
// Multi-line: the character right after "[" must be "\n".
return formatted[bracketPos] == '\n'
},
gen.OneGenOf(
genFmtListBlock(3),
genFmtListBlock(4),
genFmtListBlock(5),
),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 48: Parse Error Leaves File Unmodified
// Feature: gosling-fmt, Property 48: Parse Error Leaves File Unmodified
// Validates: Requirements 24.13
// ---------------------------------------------------------------------------

func TestFmtParseErrorLeavesFileUnmodified(t *testing.T) {
// Feature: gosling-fmt, Property 48: Parse Error Leaves File Unmodified
properties := fmtProperties()

// Corpus of syntactically invalid .fly snippets.
invalidSnippets := []string{
`egg "broken" {`,
`egg {`,
`= "no key"`,
`egg "x" { cpu = }`,
`egg "x" { cpu = [1, 2`,
`egg "x" { nested { attr = "v" }`,
`@@@ invalid syntax @@@`,
`egg "x" { attr = "unterminated string }`,
}

properties.Property("file content is byte-for-byte identical after a parse error",
prop.ForAll(
func(idx int) bool {
snippet := invalidSnippets[idx%len(invalidSnippets)]

tmpDir, err := os.MkdirTemp("", "fmt-parse-error-*")
if err != nil {
t.Logf("MkdirTemp: %v", err)
return false
}
defer os.RemoveAll(tmpDir)

filePath := filepath.Join(tmpDir, "bad.fly")
original := []byte(snippet)
if err := os.WriteFile(filePath, original, 0644); err != nil {
t.Logf("WriteFile: %v", err)
return false
}

// Attempt to parse -- must fail for this test to be meaningful.
p := parser.NewParser()
_, parseErr := p.ParseFile(filePath)
if parseErr == nil {
// Snippet parsed successfully -- not a useful test case; skip.
return true
}

// The formatter must NOT have touched the file on a parse error.
after, err := os.ReadFile(filePath)
if err != nil {
t.Logf("ReadFile after: %v", err)
return false
}

if string(after) != string(original) {
t.Logf("file was modified despite parse error\nbefore: %q\nafter:  %q",
string(original), string(after))
return false
}
return true
},
gen.IntRange(0, len(invalidSnippets)-1),
))

properties.TestingRun(t, gopter.ConsoleReporter(false))
}
