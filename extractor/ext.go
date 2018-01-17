package extractor

import (
	"path"
	"os"
	"path/filepath"
	"fmt"
)

type Package struct {
	Id          string
	Name        string
	PackagePath string
	PackageName string
	PackageDir  string
	IsPkg       bool
	IsExternal  bool
	IsStd       bool
}

type DepType int

const (
	COMP DepType = iota
	REL
)

type Dep struct {
	Id    string
	From  string
	To    string
	Type  DepType
	Count int
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

	err := filepath.Walk(pathStr, func(childPathStr string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if filepath.Ext(info.Name()) == ".go" {
			isPkg = true
		}

		if pathStr != childPathStr && info.IsDir() {
			fmt.Printf("Child path: %s\n", childPathStr)
			walkErr, childPackageList, childDepList := traverse(childPathStr)
			childPathStr := childPackageList[len(childPackageList) - 1].Id

			if walkErr != nil {
				return walkErr
			}

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

		return nil
	})

	if err != nil {
		return err, nil, nil
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