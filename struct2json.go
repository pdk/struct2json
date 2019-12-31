package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
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

	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: struct2json filename.go StructName\n")
		os.Exit(1)
	}

	dumpStruct(os.Args[1], os.Args[2])
}

func dumpStruct(filename, structName string) {

	fset := token.NewFileSet()
	mode := parser.Mode(0)

	result, err := parser.ParseFile(fset, filename, nil, mode)
	if err != nil {
		log.Fatalf("failed parsing %s: %v", filename, err)
	}

	structFields, ok := findStruct(structName, result.Decls)
	if !ok {
		log.Fatalf("struct %s not found in %s", structName, filename)
	}

	definition := Tr("name", structName)

	fields := []Tree{}

	for _, fld := range structFields.List {
		for _, name := range fld.Names {

			nxt := Tr("field", name.Name)
			nxt["type"] = handleType(fset, fld.Type)

			fields = append(fields, nxt)
		}
	}

	definition["fields"] = fields

	// switch it := decl.(type) {
	// case *ast.GenDecl:
	// 	declarations = append(declarations, handleGenDecl(fset, it))
	// case *ast.FuncDecl:
	// 	declarations = append(declarations, handleFuncDecl(fset, it))
	// default:
	// 	declarations = append(declarations, Tree{
	// 		"unhandledDeclarationType": fmt.Sprintf("%T", decl),
	// 	})
	// }

	output, err := json.MarshalIndent(definition, "", "    ")
	if err != nil {
		log.Fatalf("failed to marshal parse tree to JSON for %s: %v", filename, err)
	}

	fmt.Println(string(output))
}

func findStruct(structName string, decls []ast.Decl) (*ast.FieldList, bool) {

	for _, decl := range decls {

		structFields, ok := getStructFields(structName, decl)
		if ok {
			return structFields, true
		}
	}

	return nil, false
}

func getStructFields(structName string, declNode ast.Node) (*ast.FieldList, bool) {

	genDecl, ok := declNode.(*ast.GenDecl)
	if !ok {
		return nil, false
	}

	if len(genDecl.Specs) != 1 {
		return nil, false
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, false
	}

	if typeSpec.Name.Name != structName {
		return nil, false
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil, false
	}

	return structType.Fields, true
}

func handleGenDecl(fset *token.FileSet, it *ast.GenDecl) Tree {

	switch it.Tok {
	case token.TYPE:
		results := []Tree{}
		for _, spec := range it.Specs {
			results = append(results, handleTypeSpec(fset, spec.(*ast.TypeSpec)))
		}
		if len(results) == 1 {
			return results[0]
		}
		return Tr("type", results)
	case token.IMPORT:
		return Tr("imports", "...")
	default:
		return Tr("unhandledGenDeclToken", it.Tok.String())
	}
}

func handleTypeSpec(fset *token.FileSet, it *ast.TypeSpec) Tree {

	tree := Tree{}

	tree["name"] = it.Name.Name
	t := handleType(fset, it.Type)

	if tr, ok := t.(Tree); ok {
		tree = tree.merge(tr)
	} else {
		tree["type"] = t
	}

	return tree
}

func handleType(fset *token.FileSet, it ast.Expr) interface{} {

	switch x := it.(type) {
	case *ast.Ident:
		return x.String()
	case *ast.StructType:
		return Tr("struct", handleStructType(fset, x))
	case *ast.ArrayType:
		return Tr("array", handleType(fset, x.Elt))
	case *ast.StarExpr:
		return Tr("star", handleType(fset, x.X))
	case *ast.InterfaceType:
		return Tr("interface", handleMethods(fset, x.Methods))
	case *ast.SelectorExpr:
		return Tree{
			"pre": fmt.Sprintf("%s", handleType(fset, x.X)),
			"sel": x.Sel.String(),
		}
	case *ast.FuncType:
		return handleFuncType(fset, x)
	case *ast.MapType:
		return Tree{
			"type":  "map",
			"key":   handleType(fset, x.Key),
			"value": handleType(fset, x.Value),
		}
	default:
		return Tr("unhandledType", fmt.Sprintf("%T", it))
	}
}

func handleMethods(fset *token.FileSet, it *ast.FieldList) Tree {

	meths := []Tree{}
	composed := []Tree{}

	for _, x := range it.List {

		if f, ok := x.Type.(*ast.Ident); ok {
			composed = append(composed, Tr("name", f.Name))
			continue
		}

		if len(x.Names) == 0 {
			ast.Print(fset, x)
			continue
		}

		t := Tr("name", x.Names[0].Name)
		methType := handleType(fset, x.Type)
		if y, ok := methType.(Tree); ok {
			t = t.merge(y)
		} else {
			t["type"] = methType
		}
		meths = append(meths, t)
	}

	r := Tree{}

	if len(meths) > 0 {
		r["methods"] = meths
	}

	if len(composed) > 0 {
		r["composed"] = composed
	}

	return r
}

func handleStructType(fset *token.FileSet, it *ast.StructType) Tree {

	// ast.Print(fset, it)

	fields := []Tree{}

	for _, fld := range it.Fields.List {
		for _, name := range fld.Names {
			t := Tree{
				"name": name.Name,
				"type": handleType(fset, fld.Type),
			}
			if fld.Tag != nil {
				t["tag"] = fld.Tag.Value[1 : len(fld.Tag.Value)-1]
			}
			fields = append(fields, t)
		}
	}

	return Tr("fields", fields)
}

func handleFuncType(fset *token.FileSet, it *ast.FuncType) Tree {

	t := Tree{}

	if it.Params != nil && len(it.Params.List) > 0 {
		t["params"] = handleFieldList(fset, it.Params.List)
	}

	if it.Results != nil && len(it.Results.List) > 0 {
		t["results"] = handleFieldList(fset, it.Results.List)
	}

	return t
}

func handleFuncDecl(fset *token.FileSet, it *ast.FuncDecl) Tree {

	t := Tr("func", it.Name.Name)

	t = t.merge(handleFuncType(fset, it.Type))

	return t
}

func handleFieldList(fset *token.FileSet, it []*ast.Field) []Tree {

	fields := []Tree{}

	for _, fld := range it {

		t := Tree{}

		if len(fld.Names) > 0 {
			t["name"] = fld.Names[0].Name
		}

		t["type"] = handleType(fset, fld.Type)

		if fld.Tag != nil {
			t["tag"] = fld.Tag.Value[1 : len(fld.Tag.Value)-1]
		}

		fields = append(fields, t)
	}

	return fields
}
