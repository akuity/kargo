package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractStructFieldTagByJSONName(t *testing.T) {
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, "testdata/structs.go", nil, parser.ParseComments)
	require.NoError(t, err)

	expected := TagMap{
		"Message": map[string]string{
			"withJSONAndProtoTag": "`json:\"withJSONAndProtoTag\" protobuf:\"bytes,1,opt,name=withJSONAndProtoTag\"`",
			// Tags must be sorted
			"withUnorderedJSONAndProtoTag": "`json:\"withUnorderedJSONAndProtoTag\" protobuf:\"bytes,2,opt,name=withUnorderedJSONAndProtoTag\"`", //nolint:lll
			"withJSONTag":                  "`json:\"withJSONTag\"`",
		},
	}

	actual := make(TagMap)
	extractor := ExtractStructFieldTagByJSONName(actual)
	ast.Walk(extractor, f)
	require.Equal(t, expected, actual)
}

func TestInjectStructFieldTagByJSONName(t *testing.T) {
	// Extracts tags to be injected
	srcFS := token.NewFileSet()
	src, err := parser.ParseFile(srcFS, "testdata/generated.go", nil, parser.ParseComments)
	require.NoError(t, err)
	expectedTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(expectedTagMap), src)

	// Prepare destination file to be injected
	dstFS := token.NewFileSet()
	dst, err := parser.ParseFile(dstFS, "testdata/structs.go", nil, parser.ParseComments)
	require.NoError(t, err)

	// Ensure that tags not injected yet
	dstTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(dstTagMap), dst)
	require.NotEqual(t, dstTagMap, expectedTagMap)

	// Inject tags to dst
	ast.Walk(InjectStructFieldTagByJSONName(expectedTagMap), dst)

	// Check that tags were injected to the destination file correctly
	injectedTagMap := make(TagMap)
	ast.Walk(ExtractStructFieldTagByJSONName(injectedTagMap), dst)
	require.Equal(t, expectedTagMap, injectedTagMap)
}
