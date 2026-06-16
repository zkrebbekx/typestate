package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "typestate:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		typeList = flag.String("type", "", "comma-separated struct type names to generate builders for (required)")
		file     = flag.String("file", os.Getenv("GOFILE"), "source file containing the struct(s); defaults to $GOFILE under go:generate")
		outDir   = flag.String("dir", "", "output directory; defaults to the source file's directory")
	)
	flag.Parse()

	if *typeList == "" {
		return fmt.Errorf("-type is required (e.g. -type Client,Server)")
	}
	if *file == "" {
		return fmt.Errorf("-file is required when not run under go:generate")
	}

	src, err := os.ReadFile(*file)
	if err != nil {
		return err
	}

	dir := *outDir
	if dir == "" {
		dir = filepath.Dir(*file)
	}

	for _, typeName := range strings.Split(*typeList, ",") {
		typeName = strings.TrimSpace(typeName)
		if typeName == "" {
			continue
		}
		spec, err := parse(*file, src, typeName)
		if err != nil {
			return err
		}
		out, err := generate(spec)
		if err != nil {
			return err
		}
		outPath := filepath.Join(dir, strings.ToLower(typeName)+"_builder.go")
		if err := os.WriteFile(outPath, out, 0o644); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "typestate: wrote", outPath)
	}
	return nil
}
