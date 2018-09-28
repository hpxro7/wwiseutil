package main

import (
	"flag"
	"log"
)

const (
	shorthandSuffix = " (shorthand)"
)

type flagError string

var shouldUnpack bool
var shouldRepack bool

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
	}

	if err != "" {
		flag.Usage()
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()
	verifyFlags()
}
