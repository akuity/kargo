package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/akuity/kargo/internal/proto/codegen"
)

func extractTags(pkgDir string) codegen.TagMap {
	pkgName := path.Base(pkgDir)
	fileSet := token.NewFileSet()
	pkgs, _ := parser.ParseDir(fileSet, pkgDir, nil, parser.ParseComments)
	pkg, ok := pkgs[pkgName]
	if !ok {
		return nil
	}

	tagMap := make(codegen.TagMap)
	extractor := codegen.ExtractStructFieldTagByJSONName(tagMap)
	for _, f := range pkg.Files {
		ast.Walk(extractor, f)
	}
	return tagMap
}

func injectTags(pkgDir string, tagMap codegen.TagMap) error {
	pkgName := path.Base(pkgDir)
	fileSet := token.NewFileSet()
	pkgs, _ := parser.ParseDir(fileSet, pkgDir, func(fi fs.FileInfo) bool {
		fileName := fi.Name()
		if strings.HasSuffix(fileName, "_test.go") ||
			strings.HasSuffix(fileName, ".pb.go") {
			return false
		}
		return true
	}, parser.ParseComments)
	pkg, ok := pkgs[pkgName]
	if !ok {
		return nil
	}

	injector := codegen.InjectStructFieldTagByJSONName(tagMap)
	for fileName, f := range pkg.Files {
		ast.Walk(injector, f)
		file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC, 0)
		if err != nil {
			return errors.Wrapf(err, "open file %s", fileName)
		}
		if err := printer.Fprint(file, fileSet, f); err != nil {
			return errors.Wrapf(err, "write file %s", fileName)
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
