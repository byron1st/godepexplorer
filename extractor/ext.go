package extractor

import (
	"path"
	"os"
	"path/filepath"
	"fmt"
	"io/ioutil"
)

type Package struct {
	Id          string `json:"id"`
	Name        string `json:"label"`
	PackagePath string `json:"packagePath"`
	PackageName string `json:"packageName"`
	PackageDir  string `json:"packageDir"`
	IsPkg       bool   `json:"isPkg"`
	IsExternal  bool   `json:"isExternal"`
	IsStd       bool   `json:"isStd"`
}

type DepType int

const (
	COMP DepType = iota
	//REL
)

type Dep struct {
	Id    string  `json:"id"`
	From  string  `json:"from"`
	To    string  `json:"to"`
	Type  DepType `json:"type"`
	Count int     `json:"count"`
}

var GOPATH = path.Join(os.Getenv("GOPATH"), "src")
//var STDLIB = map[string]bool{
//	"archive":true, "bufio":true, "builtin":true, "bytes":true, "compress":true, "container":true, "context":true, "crypto":true,
//	"database":true, "debug":true, "encoding":true, "errors":true, "expvar":true, "flag":true, "fmt":true, "go":true, "hash":true,
//	"html":true, "image":true, "index":true, "io":true, "log":true, "math":true, "mime":true, "net":true, "os":true, "path":true,
//	"plugin":true, "reflect":true, "regexp":true, "runtime":true, "sort":true, "strconv":true, "strings":true, "sync":true,
//	"syscall":true, "testing":true, "text":true, "time":true, "unicode":true, "unsafe":true,
//}

func GetDirTree(rootPkgName string) (error, []*Package, []*Dep) {
	rootPkgPathStr := path.Join(GOPATH, rootPkgName)
	fmt.Printf("Root package path: %s\n", rootPkgPathStr)
	err, packageList, depList := traverse(rootPkgPathStr)
	if err != nil {
		return err, nil, nil
	}

	return nil, packageList, depList
}

func traverse(pathStr string) (error, []*Package, []*Dep) {
	packageList := make([]*Package, 0)
	depList := make([]*Dep, 0)
	goPathLen := len(GOPATH) + 1
	isPkg := false

	files, err := ioutil.ReadDir(pathStr)
	if err != nil {
		return err, nil, nil
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".go" {
			isPkg = true
		}

		childPathStr := path.Join(pathStr, file.Name())

		if file.IsDir() {
			fmt.Printf("%s - %s\n", pathStr, childPathStr)
			err, childPackageList, childDepList := traverse(childPathStr)

			if err != nil {
				return err, nil, nil
			}

			childPathStr := childPackageList[len(childPackageList) - 1].Id

			depList = append(depList, &Dep{
				Id: fmt.Sprintf("%s<>-%s", pathStr, childPathStr),
				From: childPathStr,
				To: pathStr,
				Type: COMP,
				Count: 1,
			})

			for _, childDep := range childDepList {
				depList = append(depList, childDep)
			}

			for _, childPackage := range childPackageList {
				packageList = append(packageList, childPackage)
			}
		}
	}

	_, name := path.Split(pathStr)
	packageList = append(packageList, &Package{
		Id: pathStr,
		Name: name,
		PackagePath: pathStr[goPathLen:],
		PackageDir: pathStr,
		IsPkg: isPkg,
	})

	return nil, packageList, depList
}