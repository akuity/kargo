package codegen

import (
	"fmt"
	"go/ast"
	"sort"
	"strconv"

	"github.com/fatih/structtag"
)

var (
	_ ast.Visitor = &structFieldVisitor{}
)

// structFieldVisitor is an ast.Visitor that calls given callback function
// when visiting field in a struct.
type structFieldVisitor struct {
	callback func(*ast.TypeSpec, *ast.Field)
}

func (v *structFieldVisitor) Visit(n ast.Node) ast.Visitor {
	switch node := n.(type) {
	case *ast.GenDecl:
		for _, spec := range node.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}

			for _, f := range st.Fields.List {
				v.callback(ts, f)
			}
		}
	}
	return v
}

// TagMap is a map of struct name -> field json name -> field tags
type TagMap map[string]map[string]*structtag.Tags

// ExtractStructFieldTagByJSONName returns an ast.Visitor that
// extracts struct field tag value by field's JSON name to tagMap.
// Kubebuilder struct field may have no name because it can be
// embedded in the struct (e.g. ObjectMeta). To cover this case,
// it depends on the field's JSON name, not the field name (which
// may be empty).
func ExtractStructFieldTagByJSONName(tagMap TagMap) ast.Visitor {
	return &structFieldVisitor{
		callback: func(spec *ast.TypeSpec, field *ast.Field) {
			structName := spec.Name.Name
			if _, ok := tagMap[structName]; !ok {
				tagMap[structName] = make(map[string]*structtag.Tags)
			}

			tags, err := parseFieldTags(field)
			if err != nil {
				panic(err)
			}
			key, ok := getJSONName(tags)
			if !ok {
				return
			}
			tagMap[structName][key] = tags
		},
	}
}

// InjectStructFieldTagByJSONName returns an ast.Visitor that injects
// struct field tags by field's JSON name if the given struct field's
// tag exists in tagMap.
func InjectStructFieldTagByJSONName(tagMap TagMap) ast.Visitor {
	return &structFieldVisitor{
		callback: func(spec *ast.TypeSpec, field *ast.Field) {
			if field.Tag == nil {
				return
			}

			structName := spec.Name.Name
			structTags, err := parseFieldTags(field)
			if err != nil {
				panic(err)
			}
			key, ok := getJSONName(structTags)
			if !ok {
				return
			}

			input, ok := tagMap[structName][key]
			if !ok {
				return
			}
			for idx := range input.Tags() {
				tag := input.Tags()[idx]
				// Do not override json tag
				if tag.Key == "json" {
					continue
				}
				if err := structTags.Set(tag); err != nil {
					panic(err)
				}
			}

			// Sort tags to ensure consistent output
			//
			// TODO: structtag.Tags underlying slice is not exported, so it's
			// exceptionally difficult to sort with slices.SortFunc(). Keep using
			// sort.Sort() for now.
			sort.Sort(structTags)
			field.Tag.Value = fmt.Sprintf("`%s`", structTags.String())
		},
	}
}

func parseFieldTags(field *ast.Field) (*structtag.Tags, error) {
	if field.Tag == nil {
		return nil, nil
	}

	// Remove backquotes from field tag value
	rawTag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		return nil, fmt.Errorf("unquote field tag value: %w", err)
	}
	tags, err := structtag.Parse(rawTag)
	if err != nil {
		return nil, fmt.Errorf("parse field tag: %w", err)
	}
	return tags, nil
}

func getJSONName(tags *structtag.Tags) (string, bool) {
	if tags == nil {
		return "", false
	}

	tag, err := tags.Get("json")
	if err != nil {
		return "", false
	}
	if tag.Name == "" || tag.Name == "-" {
		return "", false
	}
	return tag.Name, true
}
