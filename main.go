package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/byron1st/godepexplorer/extractor"
	"github.com/urfave/cli"
)

/**
COMMANDS: extract [package-name]
FLAGS:
--algorithm, -a	extraction algorithm (static(default), cha, rta, pointer)
--output, -o	output filepath
*/

var SupportedAlgorithms = map[string]bool{"static": true, "cha": true, "rta": true, "pointer": true}

var ErrWrongGoPackage = errors.New("wrong Go package path")
var ErrNoPackagePath = errors.New("no such Go package in your gopath")
var ErrNoSuchFileOrDir = errors.New("there is no such file or directory")
var ErrStatFileFailed = errors.New("getting stat of the given output file has been failed")
var ErrNoSuchAlgorithm = errors.New("no such algorithm")

func main() {
	app := cli.NewApp()
	app.Name = "godepexplorer"
	app.Usage = "A Go program to extract dependency relationships of a Go program"
	app.Version = "0.1.0"

	app.Commands = []cli.Command{
		buildExtractCmd(),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func buildExtractCmd() cli.Command {
	return cli.Command{
		Name:  "extract",
		Usage: "Extract all dependency relationships of a Go package",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "algorithm, a",
				Usage: "An extraction algorithm \"`ALG`\" (one of static(default), cha, rta, and pointer)",
				Value: "static",
			},
			cli.StringFlag{
				Name:  "output, o",
				Usage: "Output file path \"`FILE`\"",
				Value: ".",
			},
		},
		Action: extract,
	}
}

func extract(c *cli.Context) error {
	var pkgPath string
	var algorithm string
	var outputFilePath string

	if c.NArg() > 0 {
		pkgPath = c.Args()[0]

		info, err := os.Stat(filepath.Join(os.Getenv("GOPATH"), "src", pkgPath))

		if err != nil || !info.IsDir() {
			return ErrWrongGoPackage
		}
	} else {
		return ErrNoPackagePath
	}

	algorithm = c.String("algorithm")
	if !SupportedAlgorithms[algorithm] {
		return ErrNoSuchAlgorithm
	}

	path := c.String("output")
	outputFilePath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if info, err := os.Stat(outputFilePath); err == nil {
		if info.IsDir() {
			outputFilePath += "/DEPENDENCY.json"
		}
	} else {
		if os.IsNotExist(err) {
			return ErrNoSuchFileOrDir
		}
		return ErrStatFileFailed
	}

	fmt.Printf("Package: %s\nAlgorith: %s\nOutput: %s\n", pkgPath, algorithm, outputFilePath)

	nodes, edges, err := extractor.GetDepsWithAlgorithm(pkgPath, algorithm)
	if err != nil {
		return err
	}

	fmt.Printf("nodes len: %d, edges len: %d\n", len(nodes), len(edges))

	outputObj := struct {
		Nodes []*extractor.Pkg `json:"nodes"`
		Edges []*extractor.Dep `json:"edges"`
	}{
		Nodes: nodes,
		Edges: edges,
	}

	out, err := json.Marshal(outputObj)
	if err != nil {
		return err
	}

	f, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}

	defer f.Close()

	f.Write(out)

	return nil
}
