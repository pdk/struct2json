package main

import (
	"fmt"
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

		found := struct2json.GetStructs(goFileName)

		if len(structNames[i]) == 0 {
			// if no particular names were specified for this go file, grab them
			// all.
			doc.Append(found.Structs...)
			continue
		}

		for _, name := range structNames[i] {
			s1, ok := found.Get(name)
			if ok {
				doc.Append(s1)
			}
		}
	}

	doc.WriteJSON(os.Stdout)
}
