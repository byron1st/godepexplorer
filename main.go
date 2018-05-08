package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

/**
COMMANDS: extract
FLAGS:
--package, -p	package path
--algorithm, -a	extraction algorithm (static(default), cha, rta, pointer)
--output, -o	output filepath
*/

func main() {
	//s := server.MakeServer("localhost", 1111)
	//s.StartServer()

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
				Name:  "package, p",
				Usage: "A Go package \"`PKG`\", of which dependency relationships are extracted",
			},
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
	// If there are some command line arguments
	if c.NArg() > 0 {
		fmt.Println(c.Args()[0])
	}

	if pkgPath := c.String("package"); pkgPath != "" {
		fmt.Println(pkgPath)
	} else {
		fmt.Println("there is no package")
	}

	algorithm := c.String("algorithm")
	fmt.Println(algorithm)

	path := c.String("output")
	abspath, _ := filepath.Abs(path)
	fmt.Println(abspath)

	return nil
}
