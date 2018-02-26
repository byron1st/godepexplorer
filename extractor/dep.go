package extractor

import (
	"errors"
	"fmt"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/static"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// GetDeps extracts a list of packages and dependency relationships from a root package.
func GetDeps(pkgName string) ([]*Package, []*Dep, error) {
	program, err := buildProgram(pkgName)

	if err != nil {
		return nil, nil, err
	}

	//packageSet, depSet := inspectPackageWithCHA(program, pkgName)
	//packageSet, depSet := inspectPackageWithRTA(program, pkgName)
	packageSet, depSet := inspectPackageWithStatic(program, pkgName)
	// packageSet, depSet := inspectPackageWithPointer(program, pkgName)

	if packageSet == nil || depSet == nil {
		return nil, nil, errors.New("there is no main package")
	}

	packageList := make([]*Package, 0)
	depList := make([]*Dep, 0)

	for _, pkg := range packageSet {
		packageList = append(packageList, pkg)
	}

	for _, dep := range depSet {
		depList = append(depList, dep)
	}

	return packageList, depList, nil
}

func buildProgram(pkgName string) (*ssa.Program, error) {
	pkgPaths := []string{pkgName}
	conf := loader.Config{}
	_, err := conf.FromArgs(pkgPaths, false)
	if err != nil {
		return nil, err
	}

	load, err := conf.Load()
	if err != nil {
		return nil, err
	}

	program := ssautil.CreateProgram(load, 0)
	program.Build()
	return program, err
}

func inspectPackageWithStatic(program *ssa.Program, pkgName string) (map[string]*Package, map[string]*Dep) {
	fmt.Println("Analyze only static calls")
	return traverseCallgraph(static.CallGraph(program), pkgName)
}

func inspectPackageWithCHA(program *ssa.Program, pkgName string) (map[string]*Package, map[string]*Dep) {
	fmt.Println("Analyze using the Class Hierarchy Analysis(CHA) algorithm")
	return traverseCallgraph(cha.CallGraph(program), pkgName)
}

func inspectPackageWithRTA(program *ssa.Program, pkgName string) (map[string]*Package, map[string]*Dep) {
	fmt.Println("Analyze using the Rapid Type Analysis(RTA) algorithm")
	pkgs := program.AllPackages()

	var mains []*ssa.Package
	mains = append(mains, ssautil.MainPackages(pkgs)...)

	var roots []*ssa.Function
	for _, main := range mains {
		roots = append(roots, main.Func("init"), main.Func("main"))
	}
	cg := rta.Analyze(roots, true).CallGraph

	return traverseCallgraph(cg, pkgName)
}

func inspectPackageWithPointer(program *ssa.Program, pkgName string) (map[string]*Package, map[string]*Dep) {
	fmt.Println("Analyze using the inclusion-based Points-To Analysis algorithm")
	pkgs := program.AllPackages()

	var mains []*ssa.Package
	mains = append(mains, ssautil.MainPackages(pkgs)...)

	if len(mains) == 0 {
		return nil, nil
	}

	config := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}

	analysis, _ := pointer.Analyze(config)

	return traverseCallgraph(analysis.CallGraph, pkgName)
}

func traverseCallgraph(cg *callgraph.Graph, pkgName string) (map[string]*Package, map[string]*Dep) {
	packageSet := make(map[string]*Package)
	depSet := make(map[string]*Dep)

	callgraph.GraphVisitEdges(cg, func(e *callgraph.Edge) error {
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
		pkgObj.Meta.FuncSet[funcName] = true
		return pkgObj, funcName
	}

	newPackage := &Package{
		ID:    pkgPath,
		Label: pkg.Name(),
		Meta: &PackageMeta{
			PackagePath: pkgPath,
			PackageName: pkg.Name(),
			PackageDir:  pkgDir,
			IsExternal:  isExternal,
			IsStd:       isStd,
			IsPkg:       true,
			FuncSet:     map[string]bool{funcName: true},
		},
	}
	packageSet[newPackage.ID] = newPackage

	return newPackage, funcName
}

func addDep(depSet map[string]*Dep, callerPkg *Package, callerFuncName string, calleePkg *Package, calleeFuncName string) {
	id := getDepID(callerPkg, calleePkg)
	depObj := depSet[id]
	depAtFuncID := getDepAtFuncLevel(callerFuncName, calleeFuncName)

	if depObj != nil {
		depObj.Meta.Count++
		depObj.Meta.DepAtFuncSet[depAtFuncID] = &DepAtFunc{depAtFuncID, callerFuncName, calleeFuncName}
	} else {
		newDep := &Dep{
			ID:   id,
			From: callerPkg.ID,
			To:   calleePkg.ID,
			Meta: &DepMeta{
				Count:        1,
				DepAtFuncSet: map[string]*DepAtFunc{depAtFuncID: {depAtFuncID, callerFuncName, calleeFuncName}},
				Type:         REL,
			},
		}
		depSet[id] = newDep
	}
}

func isSynthetic(edge *callgraph.Edge) bool {
	return edge.Caller.Func.Pkg == nil || edge.Callee.Func.Synthetic != ""
}

func getPkgPath(pkg *types.Package, pkgName string) (string, string, bool, bool) {
	pkgPath := pkg.Path()
	pkgDir := path.Join(gopath, pkgPath)
	isExternal := isExternal(pkgPath)
	isStd := isStd(pkgPath)

	if isExternal && len(pkgPath) > len(pkgName) {
		pkgPath = pkgPath[strings.LastIndex(pkgPath, "/vendor/")+8:]
	} else if isStd {
		pkgDir = pkgPath
	}

	return pkgPath, pkgDir, isExternal, isStd
}

func getFuncName(functionName string, functionSig string) string {
	funcSig := functionSig[4:]
	return functionName + funcSig
}

func getDepID(callerPkg *Package, calleePkg *Package) string {
	return fmt.Sprintf("%s->%s", callerPkg.ID, calleePkg.ID)
}

func getDepAtFuncLevel(callerFuncName string, calleeFuncName string) string {
	return fmt.Sprintf("%s->%s", callerFuncName, calleeFuncName)
}

func isExternal(pkgPath string) bool {
	return strings.Contains(pkgPath, "vendor")
}

func isStd(pkgPath string) bool {
	firstPath := strings.Split(pkgPath, "/")[0]
	return stdlib[firstPath]
}
