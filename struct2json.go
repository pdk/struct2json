package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"
)

// Tree is a shortcut for map string->something
type Tree map[string]interface{}

func (t Tree) merge(other Tree) Tree {
	for k, v := range other {
		t[k] = v
	}
	return t
}

// Tr is a shortcut Tree builder
func Tr(name string, value interface{}) Tree {
	return Tree{
		name: value,
	}
}

func main() {

	goFiles := []string{}
	structNames := [][]string{}

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: struct2json f1.go structName ... f2.go structName ...\n")
		os.Exit(1)
	}

	thisGoName := ""
	theseNames := []string{}
	for _, arg := range os.Args[1:] {

		if !strings.HasSuffix(arg, ".go") {
			theseNames = append(theseNames, arg)
			continue
		}

		if thisGoName != "" {
			goFiles = append(goFiles, thisGoName)
			structNames = append(structNames, theseNames)
		}

		thisGoName = arg
		theseNames = []string{}
	}
	goFiles = append(goFiles, thisGoName)
	structNames = append(structNames, theseNames)

	results := []Tree{}

	for i, goFileName := range goFiles {
		results = append(results, getStructs(goFileName, structNames[i])...)
	}

	output, err := json.MarshalIndent(Tr("structs", results), "", "    ")
	if err != nil {
		log.Fatalf("failed to struct definition(s): %v", err)
	}

	fmt.Println(string(output))
}

func getStructs(goFileName string, structNames []string) []Tree {

	fset := token.NewFileSet()
	mode := parser.Mode(0)

	parsed, err := parser.ParseFile(fset, goFileName, nil, mode)
	if err != nil {
		log.Fatalf("failed parsing %s: %v", goFileName, err)
	}

	structs := findStructs(fset, parsed.Decls)

	result := []Tree{}

	for structName, fields := range structs {

		if len(structNames) > 0 && !contains(structNames, structName) {
			continue
		}

		definition := Tr("name", structName)
		definition = definition.merge(fields)

		result = append(result, definition)
	}

	return result
}

func contains(coll []string, item string) bool {
	for _, each := range coll {
		if each == item {
			return true
		}
	}

	return false
}

func findStructs(fset *token.FileSet, decls []ast.Decl) map[string]Tree {

	results := map[string]Tree{}

	for _, decl := range decls {

		structName, structFields, ok := getStructNameAndFields(fset, decl)
		if ok {
			results[structName] = Tr("fields", handleFieldList(fset, structFields.List))
		}
	}

	return results
}

func getStructNameAndFields(fset *token.FileSet, declNode ast.Node) (string, *ast.FieldList, bool) {

	genDecl, ok := declNode.(*ast.GenDecl)
	if !ok {
		return "", nil, false
	}

	if len(genDecl.Specs) != 1 {
		return "", nil, false
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return "", nil, false
	}

	structName := typeSpec.Name.Name
	if structName == "" {
		return "", nil, false
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return "", nil, false
	}

	return structName, structType.Fields, true
}

func handleType(fset *token.FileSet, it ast.Expr) string {

	switch x := it.(type) {
	case *ast.Ident:
		return x.String()
	case *ast.ArrayType:
		return "[]" + handleType(fset, x.Elt)
	case *ast.StarExpr:
		return "*" + handleType(fset, x.X)
	case *ast.SelectorExpr:
		return handleType(fset, x.X) + "." + x.Sel.String()
	case *ast.MapType:
		return "map[" + handleType(fset, x.Key) + "]" + handleType(fset, x.Value)
	default:
		return "unhandledType " + fmt.Sprintf("%T", it)
	}
}

func handleFieldList(fset *token.FileSet, it []*ast.Field) []Tree {

	fields := []Tree{}

	for _, fld := range it {

		t := Tree{}

		if len(fld.Names) > 0 {
			t["name"] = fld.Names[0].Name
		}

		t["type"] = handleType(fset, fld.Type)

		t = t.merge(handleFieldTag(fset, fld.Tag))

		fields = append(fields, t)
	}

	return fields
}

func handleFieldTag(fset *token.FileSet, it *ast.BasicLit) Tree {

	r := Tree{}

	if it == nil {
		return r
	}

	tag := reflect.StructTag(it.Value[1 : len(it.Value)-1])

	tagValues := Tree{}

	for _, t := range []string{
		// see https://github.com/golang/go/wiki/Well-known-struct-tags
		"xml", "json", "asn1", "reform", "dynamodb", "bigquery", "datastore", "spanner", "bson",
		"gorm", "yaml", "validate", "mapstructure", "protobuf", "db",
	} {
		v, ok := tag.Lookup(t)
		if ok {
			tagValues[t] = v
		}
	}

	if len(tagValues) > 0 {
		r["tags"] = tagValues
	}

	return r
}
