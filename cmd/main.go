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
	"github.com/hpxro7/bnkutil/pck"
	"github.com/hpxro7/bnkutil/util"
	"github.com/hpxro7/bnkutil/wwise"
)

const shorthandSuffix = " (shorthand)"
const wemExtension = ".wem"

var soundBankExtensions = []string{".nbnk", ".bnk"}
var filePackageExtensions = []string{".npck", ".pck"}

var shouldUnpack bool
var shouldReplace bool
var filePath string
var output string
var targetPath string
var verbose bool

type flagError string

func init() {
	const (
		usage    = "unpack a .bnk or .pck into seperate .wem files"
		flagName = "unpack"
	)
	flag.BoolVar(&shouldUnpack, flagName, false, usage)
	flag.BoolVar(&shouldUnpack, "u", false, shorthandDesc(flagName))
}

func init() {
	const (
		usage = "replace a set of .wem files from a source .bnk or .pck file, " +
			"outputing a fully usable .bnk or .pck with wems, offsets and lengths " +
			"updated."
		flagName = "replace"
	)
	flag.BoolVar(&shouldReplace, flagName, false, usage)
	flag.BoolVar(&shouldReplace, "r", false, shorthandDesc(flagName))
}

func init() {
	const (
		usage = "the path to the source .bnk or .pck. When unpack is used, this " +
			"is the bnk or pck file to unpack. When replace is used, this .bnk or " +
			".pck is used as a source; the wem files, offsets and lengths of this " +
			".bnk or .pck will updated and written to the file specified by output."
		flagName = "filepath"
	)
	flag.StringVar(&filePath, flagName, "", usage)
	flag.StringVar(&filePath, "f", "", shorthandDesc(flagName))
}

func init() {
	const (
		usage = "When unpack is used, this is the directory to output unpacked " +
			".wem files. When replace is used, this is the directory to output the " +
			"updated .bnk or .pck."
		flagName = "output"
	)
	flag.StringVar(&output, flagName, "", usage)
	flag.StringVar(&output, "o", "", shorthandDesc(flagName))
}

func init() {
	const (
		usage = "The directory to find .wem files in for replacing. Each wem " +
			"file's name must be a number corresponding to the index of the wem " +
			"file to replace from the source SoundBank or File Package. The index " +
			"of the first wem file is 1. The wems in the source SoundBank will be " +
			"replaced with the wems in this directory. These wems must not be " +
			"padded ahead of time; this tool will automatically add any padding " +
			"needed."
		flagName = "target"
	)
	flag.StringVar(&targetPath, flagName, "", usage)
	flag.StringVar(&targetPath, "t", "", shorthandDesc(flagName))
}

func init() {
	const (
		usage = "Shows additional information about the strcuture of the parsed " +
			"SoundBank or File Package file."
		flagName = "verbose"
	)
	flag.BoolVar(&verbose, flagName, false, usage)
	flag.BoolVar(&verbose, "v", false, shorthandDesc(flagName))
}

func shorthandDesc(flagName string) string {
	return "(shorthand for -" + flagName + ")"
}

func verifyFlags() {
	var err flagError
	switch {
	case !(shouldUnpack || shouldReplace):
		err = "Either unpack or replace should be specified"
	case shouldUnpack && shouldReplace:
		err = "Both unpack and replace cannot be specified"
	case filePath == "":
		err = "bnkpath cannot be empty"
	case output == "":
		err = "output cannot be empty"
	}

	if err != "" {
		flag.Usage()
		log.Fatal(err)
	}
}

func verifyReplaceFlags() {
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

// Verifies that the extension of the input file is supported. Returns true if
// the file is a SoundBank file and false if it is a File Package file.
func verifyInputType() bool {
	ext := filepath.Ext(filePath)
	isSoundBank := contains(soundBankExtensions, ext)
	isFilePath := contains(filePackageExtensions, ext)
	if !(isSoundBank || isFilePath) {
		flag.Usage()
		log.Fatal(ext, ", is not a supported input file type")
	}
	return isSoundBank
}

func contains(sources []string, target string) bool {
	for _, s := range sources {
		if s == target {
			return true
		}
	}
	return false
}

func unpack(isSoundBank bool) {
	var ctn wwise.Container
	var err error

	if isSoundBank {
		ctn, err = bnk.Open(filePath)
	} else { // Input is file package
		ctn, err = pck.Open(filePath)
	}
	defer ctn.Close()

	if err != nil {
		log.Fatalln("Could not parse .bnk or .pck file:", err)
	}
	if verbose {
		fmt.Println(ctn)
	}

	err = createDirIfEmpty(output)
	if err != nil {
		log.Fatalln("Could not create output directory:", err)
	}
	total := int64(0)
	for i, wem := range ctn.Wems() {
		filename := util.CanonicalWemName(i, len(ctn.Wems()))
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
	fmt.Printf("Successfully wrote %d wem(s) to %s\n", len(ctn.Wems()),
		output)
	fmt.Printf("Wrote %d bytes in total\n", total)
}

func replace(isSoundBank bool) {
	var ctn wwise.Container
	var err error

	if isSoundBank {
		ctn, err = bnk.Open(filePath)
	} else { // Input is file package
		ctn, err = pck.Open(filePath)
	}
	defer ctn.Close()

	if err != nil {
		log.Fatalln("Could not parse .bnk or .pck file:", err)
	}
	if verbose {
		fmt.Println(ctn)
	}

	targetFileInfos, err := ioutil.ReadDir(targetPath)
	if err != nil {
		log.Fatalf("Could not open target directory, \"%s\": %s\n", targetPath, err)
	}
	targets := processTargetFiles(ctn, targetFileInfos)

	ctn.ReplaceWems(targets...)

	outputFile, err := os.Create(output)
	if err != nil {
		log.Fatalf("Could not create output file \"%s\": %s\n", output, err)
	}
	total, err := ctn.WriteTo(outputFile)
	if err != nil {
		log.Fatalln("Could not write output to file: ", err)
	}
	fmt.Println("Sucessfuly replaced! Output file written to:", output)
	fmt.Printf("Wrote %d bytes in total\n", total)
}

func processTargetFiles(c wwise.Container,
	fis []os.FileInfo) []*wwise.ReplacementWem {
	var targets []*wwise.ReplacementWem
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
		if wemIndex < 0 || wemIndex >= len(c.Wems()) {
			log.Printf("Ignoring %s: This files's valid index range is "+
				"%d to %d", name, 1, len(c.Wems()))
			continue
		}
		f, err := os.Open(filepath.Join(targetPath, name))
		if err != nil {
			log.Printf("Ignoring %s: Could not open file: %s", name, err)
			continue
		}

		names = append(names, fi.Name())
		targets = append(targets, &wwise.ReplacementWem{f, wemIndex, fi.Size()})
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
	isSoundBank := verifyInputType()

	switch {
	case shouldUnpack:
		unpack(isSoundBank)
	case shouldReplace:
		verifyReplaceFlags()
		replace(isSoundBank)
	}
}
