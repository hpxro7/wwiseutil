package main

import (
	"flag"
	"fmt"
	"github.com/hpxro7/bnkutil/bnk"
	"log"
)

const (
	shorthandSuffix = " (shorthand)"
)

type flagError string

var shouldUnpack bool
var shouldRepack bool
var input string

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
		usage = "the input .bnk for unpacking or the directory to read .wem" +
			"files from for repacking"
		flagName = "input"
	)
	flag.StringVar(&input, flagName, "", usage)
	flag.StringVar(&input, "i", "", shorthandDesc(flagName))
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
	case input == "":
		err = "input cannot be empty"
	}

	if err != "" {
		flag.Usage()
		log.Fatal(err)
	}
}

func unpack() {
	file, err := bnk.Open(input)
	if err != nil {
		log.Fatalln("Could not parse .bnk file:\n", err)
	}
	fmt.Println(file)
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
