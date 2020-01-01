package struct2json

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
)

// Document contains a series of Structs
type Document struct {
	Structs []Struct `json:"structs"`
}

// Struct contains a name, and a series of Fields
type Struct struct {
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

// Field contains a name, a type, and possibly some Tags
type Field struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	Tags map[string]string `json:"tags,omitempty"`
}

// GetStructs parses a single .go file, finding all declared structs. If
// structNames is empty, all structs are returned. If structNames is not empty,
// then only those named are returned.
func GetStructs(goFileName string, structNames []string) []Struct {

	fset := token.NewFileSet()
	mode := parser.Mode(0)

	parsed, err := parser.ParseFile(fset, goFileName, nil, mode)
	if err != nil {
		log.Fatalf("failed parsing %s: %v", goFileName, err)
	}

	structs := findStructs(fset, parsed.Decls)

	result := []Struct{}

	for structName, Struct := range structs {

		if len(structNames) > 0 && !contains(structNames, structName) {
			continue
		}

		result = append(result, Struct)
	}

	return result
}

func findStructs(fset *token.FileSet, decls []ast.Decl) map[string]Struct {

	results := map[string]Struct{}

	for _, decl := range decls {

		structName, structFields, ok := getStructNameAndFields(fset, decl)
		if ok {
			results[structName] = Struct{
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

func handleFieldList(fset *token.FileSet, it []*ast.Field) []Field {

	fields := []Field{}

	for _, fld := range it {

		t := Field{
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

func contains(coll []string, item string) bool {
	for _, each := range coll {
		if each == item {
			return true
		}
	}

	return false
}
