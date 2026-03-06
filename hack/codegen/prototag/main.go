// `prototag` is a tool to manage `protobuf` tags in the Kubebuilder structs.
//
// It extracts `protobuf` tags from the `buf` generated Go files and
// inject to the Kubebuilder structs.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"

	"github.com/akuity/kargo/pkg/proto/codegen"
)

func extractTags(pkgDir string) codegen.TagMap {
	pkgName := path.Base(pkgDir)
	fileSet := token.NewFileSet()
	entries, _ := os.ReadDir(pkgDir)

	tagMap := make(codegen.TagMap)
	extractor := codegen.ExtractStructFieldTagByJSONName(tagMap)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		f, err := parser.ParseFile(fileSet, path.Join(pkgDir, entry.Name()), nil, parser.ParseComments)
		if err != nil || f.Name.Name != pkgName {
			continue
		}
		ast.Walk(extractor, f)
	}
	return tagMap
}

func injectTags(pkgDir string, tagMap codegen.TagMap) error {
	pkgName := path.Base(pkgDir)
	fileSet := token.NewFileSet()
	entries, _ := os.ReadDir(pkgDir)

	injector := codegen.InjectStructFieldTagByJSONName(tagMap)
	for _, entry := range entries {
		fileName := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(fileName, ".go") ||
			strings.HasSuffix(fileName, "_test.go") ||
			strings.HasSuffix(fileName, ".pb.go") {
			continue
		}
		filePath := path.Join(pkgDir, fileName)
		f, err := parser.ParseFile(fileSet, filePath, nil, parser.ParseComments)
		if err != nil || f.Name.Name != pkgName {
			continue
		}
		ast.Walk(injector, f)
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return fmt.Errorf("open file %s: %w", filePath, err)
		}
		if err := format.Node(file, fileSet, f); err != nil {
			return fmt.Errorf("write file %s: %w", filePath, err)
		}
	}
	return nil
}

func main() {
	var srcDir, dstDir string
	flag.StringVar(&srcDir, "src-dir", "", "path to the source directory (e.g. pkg/api/v1alpha1)")
	flag.StringVar(&dstDir, "dst-dir", "", "path to the destination directory (e.g. api/v1alpha1)")
	flag.Parse()

	if srcDir == "" {
		fmt.Fprintln(os.Stderr, "src-dir should not be empty")
		os.Exit(1)
	}
	if dstDir == "" {
		fmt.Fprintln(os.Stderr, "dst-dir should not be empty")
		os.Exit(1)
	}

	tagMap := extractTags(srcDir)
	if err := injectTags(dstDir, tagMap); err != nil {
		panic(err)
	}
}
