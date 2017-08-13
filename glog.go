// Glog is a very, very simple static site generator.
package main

import (
	"flag"
	"fmt"
	"github.com/russross/blackfriday"
	"text/template"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
)

func main() {
	outdir := flag.String("o", "", "Specify an output directory for .html files (i.e., not the input directory).")
	fglob := flag.String("g", "*.txt", "Specify the file glob pattern of input files. Defaults to '*.txt'.")
	flag.Parse()
	var inDir, outDir string
	var err error
	switch len(flag.Args()) {
	case 0:
		inDir, err = os.Getwd()
		if err != nil {
			log.Fatal("Couldn't get the current directory for input: ", err)
		}
		outDir = inDir
	case 1:
		inDir = flag.Args()[0]
		if _, err := os.Stat(inDir); os.IsNotExist(err) {
			log.Fatal("Input directory error: ", err)
		}
		outDir = inDir
	default:
		log.Fatalf("One or zero arguments required, but %d supplied.", len(flag.Args()))
	}
	if *outdir != "" {
		if _, err := os.Stat(*outdir); os.IsNotExist(err) {
			log.Fatal("Output directory error: ", err)
		}
		outDir = *outdir
	}
	tmpl, err := template.ParseFiles(path.Join(inDir, "template.html"))
	if err != nil {
		panic(err)
	}
	inFiles, err := filepath.Glob(path.Join(inDir, *fglob))
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range inFiles {
		input, err := ioutil.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}
		Body := string(blackfriday.MarkdownCommon(input))
		err = tmpl.Execute(os.Stdout, Body)
	}
	fmt.Println(outDir)
}
