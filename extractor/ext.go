package extractor

import (
	"path"
	"os"
	"path/filepath"
	"fmt"
	"io/ioutil"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph"
	"strings"
	"go/types"
)

type Package struct {
	Id          string          `json:"id"`
	Name        string          `json:"label"`
	PackagePath string          `json:"packagePath"`
	PackageName string          `json:"packageName"`
	PackageDir  string          `json:"packageDir"`
	IsPkg       bool            `json:"isPkg"`
	IsExternal  bool            `json:"isExternal"`
	IsStd       bool            `json:"isStd"`
	FuncSet     map[string]bool `json:"funcSet"`
}

type DepType int

const (
	COMP DepType = iota
	REL
)

type Dep struct {
	Id        string          `json:"id"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Type      DepType         `json:"type"`
	Count     int             `json:"count"`
	DepAtFunc map[string]bool `json:"depAtFunc"`
}

var GOPATH = path.Join(os.Getenv("GOPATH"), "src")
var STDLIB = map[string]bool{
	"archive":true, "bufio":true, "builtin":true, "bytes":true, "compress":true, "container":true, "context":true, "crypto":true,
	"database":true, "debug":true, "encoding":true, "errors":true, "expvar":true, "flag":true, "fmt":true, "go":true, "hash":true,
	"html":true, "image":true, "index":true, "io":true, "log":true, "math":true, "mime":true, "net":true, "os":true, "path":true,
	"plugin":true, "reflect":true, "regexp":true, "runtime":true, "sort":true, "strconv":true, "strings":true, "sync":true,
	"syscall":true, "testing":true, "text":true, "time":true, "unicode":true, "unsafe":true,
}

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

func GetDeps(pkgName string) (error, []*Package, []*Dep) {
	err, program := buildProgram(pkgName)

	if err != nil {
		return err, nil, nil
	}

	packageSet, depSet := inspectPackageWithCha(program, pkgName)

	packageList := make([]*Package, 0)
	depList := make([]*Dep, 0)

	for _, pkg := range packageSet {
		packageList = append(packageList, pkg)
	}

	f, _ := os.Create("output.csv")
	defer f.Close()
	for _, dep := range depSet {
		depList = append(depList, dep)

		f.WriteString(dep.Id + "\n")
	}
	f.Sync()

	return nil, packageList, depList
}

func buildProgram(pkgName string) (error, *ssa.Program) {
	pkgPaths := []string{pkgName}
	conf := loader.Config{}
	s, err := conf.FromArgs(pkgPaths, false)
	if err != nil {
		println(err.Error())
		println(s)
	}
	load, err := conf.Load()

	if err != nil {
		println(err.Error())
	}
	program := ssautil.CreateProgram(load, 0)
	program.Build()
	return err, program
}

func inspectPackageWithCha(program *ssa.Program, pkgName string) (map[string]*Package, map[string]*Dep) {
	packageSet := make(map[string]*Package)
	depSet := make(map[string]*Dep)

	callgraph.GraphVisitEdges(cha.CallGraph(program), func(e *callgraph.Edge) error {
		if isSynthetic(e) {
			return nil
		}

		if e.Caller.Func.Pkg.Pkg.Path() != pkgName {
			return nil
		}

		callerPkg, callerFuncName := addPackage(packageSet, e.Caller, pkgName)
		calleePkg, calleeFuncName := addPackage(packageSet, e.Callee, pkgName)

		addDep(depSet, callerPkg, callerFuncName, calleePkg, calleeFuncName)

		return nil
	})

	return packageSet, depSet
}

func addPackage(packageSet map[string]*Package, n *callgraph.Node, pkgName string) (*Package, string) {
	pkg := n.Func.Pkg.Pkg
	pkgPath, pkgDir, isExternal, isStd := getPkgPath(pkg, pkgName)

	funcName := getFuncName(n.Func.Name(), n.Func.Signature.String())

	pkgObj := packageSet[pkgPath]

	if pkgObj != nil {
		pkgObj.FuncSet[funcName] = true
		return pkgObj, funcName
	}

	newPackage := &Package{
		Id:          pkgPath,
		Name:        pkg.Name(),
		PackagePath: pkgPath,
		PackageName: pkg.Name(),
		PackageDir:  pkgDir,
		IsExternal:  isExternal,
		IsStd:       isStd,
		IsPkg:       true,
		FuncSet:     map[string]bool{funcName: true},
	}
	packageSet[newPackage.Id] = newPackage

	return newPackage, funcName
}

func addDep(depSet map[string]*Dep, callerPkg *Package, callerFuncName string, calleePkg *Package, calleeFuncName string) {
	id := getDepId(callerPkg, calleePkg)
	depObj := depSet[id]
	depAtFuncLevel := getDepAtFuncLevel(callerFuncName, calleeFuncName)

	if depObj != nil {
		depObj.Count++
		depObj.DepAtFunc[depAtFuncLevel] = true
	} else {
		newDep := &Dep{
			Id:    id,
			From:  callerPkg.Id,
			To:    calleePkg.Id,
			Count: 1,
			DepAtFunc: map[string]bool{depAtFuncLevel: true},
			Type: REL,
		}
		depSet[id] = newDep
	}
}

func isSynthetic (edge *callgraph.Edge) bool {
	return edge.Caller.Func.Pkg == nil || edge.Callee.Func.Synthetic != ""
}

func getPkgPath (pkg *types.Package, pkgName string) (string, string, bool, bool) {
	pkgPath := pkg.Path()
	pkgDir := path.Join(GOPATH, pkgPath)
	isExternal := strings.Contains(pkgPath, "vendor") || !strings.Contains(pkgPath, pkgName) // TODO: vendor 체크가 Path()에서 왜 필요한지?
	isStd := STDLIB[pkg.Name()]

	if isExternal && len(pkgPath) > len(pkgName){
		pkgPath = pkgPath[strings.LastIndex(pkgPath, "/vendor/") + 8:]
	} else if isStd {
		pkgDir = pkgPath
	}

	return pkgPath, pkgDir, isExternal, isStd
}

func getFuncName(functionName string, functionSig string) string {
	funcSig := functionSig[4:]
	return functionName + funcSig
}

func getDepId(callerPkg *Package, calleePkg *Package) string {
	return fmt.Sprintf("%s->%s", callerPkg.Id, calleePkg.Id)
}

func getDepAtFuncLevel(callerFuncName string, calleeFuncName string) string {
	return fmt.Sprintf("%s->%s", callerFuncName, calleeFuncName)
}
