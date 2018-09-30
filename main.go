package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

import (
	"github.com/hpxro7/bnkutil/bnk"
)

const shorthandSuffix = " (shorthand)"

type flagError string

var shouldUnpack bool
var shouldRepack bool
var bnkPath string
var output string

func init() {
	const (
		usage    = "unpack a .bnk into seperate .wem files"
		flagName = "unpack"
	)
	flag.BoolVar(&shouldUnpack, flagName, false, usage)
	flag.BoolVar(&shouldUnpack, "u", false, shorthandDesc(flagName))
}

func init() {
	const (
		usage    = "repack a set of .wem files into a .bnk file"
		flagName = "repack"
	)
	flag.BoolVar(&shouldRepack, flagName, false, usage)
	flag.BoolVar(&shouldRepack, "r", false, shorthandDesc(flagName))
}

func init() {
	const (
		usage = "the path to the target .bnk. When unpack is used, this is the " +
			"bnk file to unpack. When repack is used, this is the template bnk " +
			"used; wem files will be replaced using this bnk."
		flagName = "bnkpath"
	)
	flag.StringVar(&bnkPath, flagName, "", usage)
	flag.StringVar(&bnkPath, "b", "", shorthandDesc(flagName))
}

func init() {
	const (
		usage = "The directory to output .wem files for unpacking or the" +
			"directory to output the combined .bnk file for repacking."
		flagName = "output"
	)
	flag.StringVar(&output, flagName, "", usage)
	flag.StringVar(&output, "o", "", shorthandDesc(flagName))
}

func shorthandDesc(flagName string) string {
	return "(shorthand for -" + flagName + ")"
}

func verifyFlags() {
	var err flagError
	switch {
	case !(shouldUnpack || shouldRepack):
		err = "Either unpack or repack should be specified"
	case shouldUnpack && shouldRepack:
		err = "Both unpack and repack cannot be specified"
	case bnkPath == "":
		err = "bnkpath cannot be empty"
	case output == "":
		err = "output cannot be empty"
	}

	if err != "" {
		flag.Usage()
		log.Fatal(err)
	}
}

func unpack() {
	bnk, err := bnk.Open(bnkPath)
	defer bnk.Close()
	if err != nil {
		log.Fatalln("Could not parse .bnk file:\n", err)
	}
	fmt.Println(bnk)

	err = createDirIfEmpty(output)
	if err != nil {
		log.Fatalln("Could not create output directory:", err)
	}
	total := int64(0)
	for i, wem := range bnk.DataSection.Wems {
		filename := fmt.Sprintf("%03d.wem", i+1)
		f, err := os.Create(filepath.Join(output, filename))
		if err != nil {
			log.Fatalf("Could not create wem file \"%s\": %s", filename, err)
		}
		n, err := io.Copy(f, wem)
		if err != nil {
			log.Fatalf("Could not write wem file \"%s\": %s", filename, err)
		}
		total += n
	}
	fmt.Println("Total bytes written: ", total)
}

func createDirIfEmpty(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.Mkdir(output, os.ModePerm)
	}
	return nil
}

func main() {
	flag.Parse()
	verifyFlags()

	switch {
	case shouldUnpack:
		unpack()
	case shouldRepack:
		log.Fatal("Repack is currently unimplemented")
	}
}
