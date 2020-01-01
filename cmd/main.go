package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pdk/struct2json"
)

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

	doc := struct2json.Document{}

	for i, goFileName := range goFiles {
		doc.Structs = append(doc.Structs, struct2json.GetStructs(goFileName, structNames[i])...)
	}

	output, err := json.MarshalIndent(doc, "", "    ")
	if err != nil {
		log.Fatalf("failed to struct definition(s): %v", err)
	}

	fmt.Println(string(output))
}
