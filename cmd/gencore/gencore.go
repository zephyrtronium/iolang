package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, os.Args[0], "output.go iofiles/ iofiles/ ...")
		os.Exit(1)
	}
	out, err := os.Create(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if _, err = out.WriteString("package iolang\n\n// Code generated by gencore; DO NOT EDIT\n\nvar coreIo = [][]byte{\n"); err != nil {
		panic(err)
	}
	var paths []string
	for _, dir := range os.Args[2:] {
		fis, err := ioutil.ReadDir(dir)
		if err != nil {
			panic(err)
		}
		for _, fi := range fis {
			if fi.IsDir() {
				continue
			}
			path := filepath.Join(dir, fi.Name())
			data := enc(path)
			if _, err = fmt.Fprintf(out, "\t%#v,\n", data); err != nil {
				panic(err)
			}
			paths = append(paths, path)
		}
	}
	if _, err = fmt.Fprintf(out, "}\n\nvar coreFiles = %#v\n", paths); err != nil {
		panic(err)
	}
}

func enc(path string) []byte {
	in, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer in.Close()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err = io.Copy(w, in); err != nil {
		panic(err)
	}
	w.Close()
	return buf.Bytes()
}
