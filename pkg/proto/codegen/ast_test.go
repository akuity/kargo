package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/fatih/structtag"
	"github.com/stretchr/testify/require"
)

func TestExtractStructFieldTagByJSONName(t *testing.T) {
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, "testdata/structs.go", nil, parser.ParseComments)
	require.NoError(t, err)

	expected := TagMap{
		"Message": map[string]*structtag.Tags{
			"withJSONAndProtoTag": mustParseStructTags(
				`json:"withJSONAndProtoTag" protobuf:"bytes,1,opt,name=withJSONAndProtoTag"`),
			"withJSONOmitEmptyAndProtoTag": mustParseStructTags(
				`json:"withJSONOmitEmptyAndProtoTag,omitempty" protobuf:"bytes,2,opt,name=withUnorderedJSONAndProtoTag"`),
			"withJSONTag":          mustParseStructTags(`json:"withJSONTag"`),
			"withJSONOmitEmptyTag": mustParseStructTags(`json:"withJSONOmitEmptyTag,omitempty"`),
		},
	}

	actual := make(TagMap)
	extractor := ExtractStructFieldTagByJSONName(actual)
	ast.Walk(extractor, f)
	equalTagMap(t, expected, actual)
}

func TestInjectStructFieldTagByJSONName(t *testing.T) {
	// Extracts tags to be injected
	srcFS := token.NewFileSet()
	src, err := parser.ParseFile(srcFS, "testdata/generated.go", nil, parser.ParseComments)
	require.NoError(t, err)
	srcTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(srcTagMap), src)

	// Prepare destination file to be injected
	dstFS := token.NewFileSet()
	dst, err := parser.ParseFile(dstFS, "testdata/structs.go", nil, parser.ParseComments)
	require.NoError(t, err)

	// Inject tags to dst
	ast.Walk(InjectStructFieldTagByJSONName(srcTagMap), dst)
	actualTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(actualTagMap), dst)

	// Validate if tags are injected correctly
	expectedFS := token.NewFileSet()
	expected, err := parser.ParseFile(expectedFS, "testdata/expected.go", nil, parser.ParseComments)
	require.NoError(t, err)
	expectedTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(expectedTagMap), expected)
	equalTagMap(t, expectedTagMap, actualTagMap)
}

func mustParseStructTags(input string) *structtag.Tags {
	t, err := structtag.Parse(input)
	if err != nil {
		panic(err)
	}
	return t
}

func equalTagMap(t *testing.T, expected, actual TagMap) {
	require.Len(t, actual, len(expected))
	for k, v := range expected {
		require.Contains(t, actual, k)
		require.Len(t, actual[k], len(v))
		for fk, fv := range v {
			require.Contains(t, actual[k], fk)
			require.EqualValues(t, fv.Tags(), actual[k][fk].Tags())
		}
	}
}
