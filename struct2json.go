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

// StructsDocument contains a series of StructDefs
type StructsDocument struct {
	Structs []StructDef `json:"structs"`
}

// StructDef contains a name, and a series of FieldDefs
type StructDef struct {
	Name   string     `json:"name"`
	Fields []FieldDef `json:"fields"`
}

// FieldDef contains a name, a type, and possibly some Tags
type FieldDef struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	Tags map[string]string `json:"tags,omitempty"`
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

	doc := StructsDocument{}

	for i, goFileName := range goFiles {
		doc.Structs = append(doc.Structs, getStructs(goFileName, structNames[i])...)
	}

	output, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		log.Fatalf("failed to struct definition(s): %v", err)
	}

	fmt.Println(string(output))
}

func getStructs(goFileName string, structNames []string) []StructDef {

	fset := token.NewFileSet()
	mode := parser.Mode(0)

	parsed, err := parser.ParseFile(fset, goFileName, nil, mode)
	if err != nil {
		log.Fatalf("failed parsing %s: %v", goFileName, err)
	}

	structs := findStructs(fset, parsed.Decls)

	result := []StructDef{}

	for structName, structDef := range structs {

		if len(structNames) > 0 && !contains(structNames, structName) {
			continue
		}

		result = append(result, structDef)
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

func findStructs(fset *token.FileSet, decls []ast.Decl) map[string]StructDef {

	results := map[string]StructDef{}

	for _, decl := range decls {

		structName, structFields, ok := getStructNameAndFields(fset, decl)
		if ok {
			results[structName] = StructDef{
				Name:   structName,
				Fields: handleFieldList(fset, structFields.List),
			}
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

func handleFieldList(fset *token.FileSet, it []*ast.Field) []FieldDef {

	fields := []FieldDef{}

	for _, fld := range it {

		t := FieldDef{
			Type: handleType(fset, fld.Type),
			Tags: handleFieldTag(fset, fld.Tag),
		}

		if len(fld.Names) > 0 {
			t.Name = fld.Names[0].Name
		} else {
			t.Name = t.Type
		}

		fields = append(fields, t)
	}

	return fields
}

func handleFieldTag(fset *token.FileSet, it *ast.BasicLit) map[string]string {

	result := map[string]string{}

	if it == nil {
		return result
	}

	tag := reflect.StructTag(it.Value[1 : len(it.Value)-1])

	for _, t := range []string{
		// see https://github.com/golang/go/wiki/Well-known-struct-tags
		"xml", "json", "asn1", "reform", "dynamodb", "bigquery", "datastore", "spanner", "bson",
		"gorm", "yaml", "validate", "mapstructure", "protobuf", "db",
	} {
		v, ok := tag.Lookup(t)
		if ok {
			result[t] = v
		}
	}

	return result
}
