package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

import (
	"github.com/hpxro7/bnkutil/bnk"
)

const shorthandSuffix = " (shorthand)"
const wemExtension = ".wem"

var shouldUnpack bool
var shouldRepack bool
var bnkPath string
var output string
var targetPath string

type flagError string
type targetWem struct {
	*os.File
	WemIndex int
	FileSize int64
}

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
		usage    = "repack and replace a set of .wem files into a source .bnk file"
		flagName = "repack"
	)
	flag.BoolVar(&shouldRepack, flagName, false, usage)
	flag.BoolVar(&shouldRepack, "r", false, shorthandDesc(flagName))
}

func init() {
	const (
		usage = "the path to the source .bnk. When unpack is used, this is the " +
			"bnk file to unpack. When repack is used, this is the template bnk " +
			"used; wem files will be replaced using this bnk as a source."
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

func init() {
	const (
		usage = "The directory to find .wem files in for replacing. Each wem " +
			"file's name must be a number corresponding to the index of the wem " +
			"file to replace from the source SoundBank. The index of the first wem " +
			"file is 1. The wems in the source SoundBank will be replaced with the " +
			"wems in this directory."
		flagName = "target"
	)
	flag.StringVar(&targetPath, flagName, "", usage)
	flag.StringVar(&targetPath, "t", "", shorthandDesc(flagName))
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

func verifyRepackFlags() {
	var err flagError
	switch {
	case targetPath == "":
		err = "target cannot be empty"
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

	err = createDirIfEmpty(output)
	if err != nil {
		log.Fatalln("Could not create output directory:", err)
	}
	total := int64(0)
	for i, wem := range bnk.DataSection.Wems {
		// Wems are indexed internally starting from 0, but the file names start
		// at 1.
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
	fmt.Printf("Successfully wrote %d wem(s) to %s\n", len(bnk.DataSection.Wems),
		output)
	fmt.Printf("Wrote %d bytes in total\n", total)
}

func repack() {
	bnk, err := bnk.Open(bnkPath)
	defer bnk.Close()
	if err != nil {
		log.Fatalln("Could not parse .bnk file:", err)
	}

	outputFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatalf("Could not open file \"%s\" for writing: %s\n", output, err)
	}

	targetFileInfos, err := ioutil.ReadDir(targetPath)
	if err != nil {
		log.Fatalf("Could not open target directory, \"%s\": %s\n", targetPath, err)
	}
	targets := processTargetFiles(bnk, targetFileInfos)

	for _, t := range targets {
		bnk.ReplaceWem(t, t.WemIndex, t.FileSize)
	}

	total, err := bnk.WriteTo(outputFile)
	if err != nil {
		log.Fatalln("Could not write SoundBank to file: ", err)
	}
	fmt.Println("Sucessfuly repacked SoundBank file to:", output)
	fmt.Printf("Wrote %d bytes in total\n", total)
}

func processTargetFiles(bnk *bnk.File, fis []os.FileInfo) []*targetWem {
	var targets []*targetWem
	var names []string
	for _, fi := range fis {
		name := fi.Name()
		ext := filepath.Ext(name)
		if ext != wemExtension {
			log.Printf("Ignoring %s: It does not have a .wem file extension",
				name)
			continue
		}
		wemIndex, err := strconv.Atoi(strings.TrimSuffix(name, ext))
		// Wems are indexed internally starting from 0, but the file names start
		// at 1.
		wemIndex--
		if err != nil {
			log.Printf("Ignoring %s: It does not have a valid integer name",
				name)
			continue
		}
		if wemIndex < 0 || wemIndex >= bnk.IndexSection.WemCount {
			log.Printf("Ignoring %s: This SoundBank's valid index range is "+
				"%d to %d", name, 1, bnk.IndexSection.WemCount)
			continue
		}
		f, err := os.Open(filepath.Join(targetPath, name))
		if err != nil {
			log.Printf("Ignoring %s: Could not open file: %s", name, err)
			continue
		}

		names = append(names, fi.Name())
		targets = append(targets, &targetWem{f, wemIndex, fi.Size()})
	}
	if len(targets) == 0 {
		log.Fatal("There are no replacement wems")
	}
	fmt.Printf("Using %d replacement wem(s): %s\n", len(targets),
		strings.Join(names, ", "))
	return targets
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
		verifyRepackFlags()
		repack()
	}
}
