package extractor

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
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
func GetDeps(pkgName string) ([]*Pkg, []*Dep, error) {
	// allMains := traverseSubDir(path.Join(gopath, pkgName))
	// for _, main := range allMains {
	// 	fmt.Println(main)
	// }

	program, err := buildProgram(pkgName)

	if err != nil {
		return nil, nil, err
	}

	pkgSet, depSet := inspectPackageWithCHA(program, pkgName)
	// pkgSet, depSet := inspectPackageWithRTA(program, pkgName)
	// pkgSet, depSet := inspectPackageWithStatic(program, pkgName)
	// pkgSet, depSet := inspectPackageWithPointer(program, pkgName)

	if pkgSet == nil || depSet == nil {
		return nil, nil, errors.New("there is no main package")
	}

	pkgList := make([]*Pkg, 0)
	depList := make([]*Dep, 0)

	for _, pkg := range pkgSet {
		pkgList = append(pkgList, pkg)
	}

	for _, dep := range depSet {
		depList = append(depList, dep)
	}

	return pkgList, depList, nil
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

func inspectPackageWithStatic(program *ssa.Program, pkgName string) (map[string]*Pkg, map[string]*Dep) {
	fmt.Println("Analyze only static calls")
	packageSet, depSet := traverseCallgraph(static.CallGraph(program), pkgName)

	return constructTree(packageSet, depSet)
}

func inspectPackageWithCHA(program *ssa.Program, pkgName string) (map[string]*Pkg, map[string]*Dep) {
	fmt.Println("Analyze using the Class Hierarchy Analysis(CHA) algorithm")
	return traverseCallgraph(cha.CallGraph(program), pkgName)
}

func inspectPackageWithRTA(program *ssa.Program, pkgName string) (map[string]*Pkg, map[string]*Dep) {
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

func inspectPackageWithPointer(program *ssa.Program, pkgName string) (map[string]*Pkg, map[string]*Dep) {
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

func traverseCallgraph(cg *callgraph.Graph, pkgName string) (map[string]*Pkg, map[string]*Dep) {
	pkgSet := make(map[string]*Pkg)
	depSet := make(map[string]*Dep)

	callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		if isSynthetic(edge) {
			return nil
		}

		// Remove an edge if packages of its caller and callee are same
		if getPkgPath(edge.Caller, pkgName) == getPkgPath(edge.Callee, pkgName) {
			return nil
		}

		addPkg(pkgSet, edge.Caller, pkgName)
		addPkg(pkgSet, edge.Callee, pkgName)
		addDep(depSet, edge, pkgName, pkgSet)

		return nil
	})

	return pkgSet, depSet
}

func constructTree(packageSet map[string]*Pkg, depSet map[string]*Dep) (map[string]*Pkg, map[string]*Dep) {
	for pkgID, pkg := range packageSet {
		pkgStringTokens := strings.Split(pkg.ID, "/")
		if len(pkgStringTokens) != 1 {
			parentPkgID := strings.Join(pkgStringTokens[:len(pkgStringTokens)-1], "/")
			if packageSet[parentPkgID] != nil {
				packageSet[parentPkgID].Meta.Children[pkgID] = true
				pkg.Meta.Parent = parentPkgID

				compDep := getCompDep(parentPkgID, pkgID)
				depSet[compDep.ID] = compDep
			}
		}
	}

	return packageSet, depSet
}

func addPkg(pkgSet map[string]*Pkg, node *callgraph.Node, pkgName string) {
	pkgPath := getPkgPath(node, pkgName)
	if pkgObj := pkgSet[getPkgIDFromPath(pkgPath)]; pkgObj == nil {
		newPkg := &Pkg{
			ID:    getPkgIDFromPath(pkgPath),
			Label: getPkgName(node),
			Meta: &PkgMeta{
				PkgPath:         pkgPath,
				PkgName:         getPkgName(node),
				PkgDir:          getPkgDirFromPath(pkgPath),
				PkgType:         getPkgTypeFromPath(pkgPath),
				SourceEdgeIDSet: make(map[string]bool),
				SinkEdgeIDSet:   make(map[string]bool),
				Parent:          "",
				Children:        make(map[string]bool),
			},
		}

		pkgSet[newPkg.ID] = newPkg
	}
}

func addDep(depSet map[string]*Dep, edge *callgraph.Edge, pkgName string, pkgSet map[string]*Pkg) {
	depID := getDepID(edge, pkgName)
	depAtFunc := getDepAtFunc(edge, pkgName)

	if depObj := depSet[depID]; depObj == nil {
		newDep := &Dep{
			ID:   depID,
			From: getPkgIDFromPath(getPkgPath(edge.Caller, pkgName)),
			To:   getPkgIDFromPath(getPkgPath(edge.Callee, pkgName)),
			Meta: &DepMeta{
				DepAtFuncSet: map[string]*DepAtFunc{depAtFunc.ID: depAtFunc},
				Type:         REL,
			},
		}

		depSet[depID] = newDep
	} else {
		depObj.Meta.DepAtFuncSet[depAtFunc.ID] = depAtFunc
	}

	if callerPkg := pkgSet[getPkgIDFromPath(getPkgPath(edge.Caller, pkgName))]; callerPkg != nil {
		callerPkg.Meta.SourceEdgeIDSet[depID] = true
	}

	if calleePkg := pkgSet[getPkgIDFromPath(getPkgPath(edge.Callee, pkgName))]; calleePkg != nil {
		calleePkg.Meta.SinkEdgeIDSet[depID] = true
	}
}

func getDepAtFunc(edge *callgraph.Edge, pkgName string) *DepAtFunc {
	return &DepAtFunc{
		ID:   getDepAtFuncID(edge, pkgName),
		From: getFunc(edge.Caller),
		To:   getFunc(edge.Callee),
	}
}

func getCompDep(parentPkgID string, childPkgID string) *Dep {
	return &Dep{
		ID:   getCompDepID(parentPkgID, childPkgID),
		From: parentPkgID,
		To:   childPkgID,
		Meta: &DepMeta{
			Type:         COMP,
			DepAtFuncSet: make(map[string]*DepAtFunc),
		},
	}
}

func isSynthetic(edge *callgraph.Edge) bool {
	return edge.Caller.Func.Pkg == nil || edge.Callee.Func.Synthetic != ""
}

func getPkgName(node *callgraph.Node) string {
	return node.Func.Pkg.Pkg.Name()
}

func getPkgPath(node *callgraph.Node, pkgName string) string {
	pkgPath := node.Func.Pkg.Pkg.Path()
	if isExt(pkgPath) && len(pkgPath) > len(pkgName) {
		// TODO: /Godeps/_workspace/ 도 /vendor/ 와 동일하게 처리해주어야 함.
		return pkgPath[strings.LastIndex(pkgPath, "/vendor/")+8:]
	}
	return pkgPath
}

func getPkgDirFromPath(pkgPath string) string {
	pkgDir := path.Join(gopath, pkgPath)

	if isStd(pkgPath) {
		return pkgPath
	}

	return pkgDir
}

func getPkgTypeFromPath(pkgPath string) PkgType {
	if isExt(pkgPath) {
		return EXT
	} else if isStd(pkgPath) {
		return STD
	} else {
		return NOR
	}
}

func getFunc(node *callgraph.Node) string {
	funcName := node.Func.Name()
	funcSig := node.Func.Signature.String()[4:]

	return fmt.Sprint(funcName, funcSig)
}

func getPkgIDFromPath(pkgPath string) string {
	return hashByMD5(pkgPath)
}

func getDepID(edge *callgraph.Edge, pkgName string) string {
	callerPkgID := getPkgIDFromPath(getPkgPath(edge.Caller, pkgName))
	calleePkgID := getPkgIDFromPath(getPkgPath(edge.Callee, pkgName))

	return hashByMD5(fmt.Sprintf("%s->%s", callerPkgID, calleePkgID))
}

func getCompDepID(parentPkgID string, childPkgID string) string {
	return hashByMD5(fmt.Sprintf("%s<>-%s", parentPkgID, childPkgID))
}

func getDepAtFuncID(edge *callgraph.Edge, pkgName string) string {
	callerFuncName := getFunc(edge.Caller)
	calleeFuncName := getFunc(edge.Callee)

	return hashByMD5(fmt.Sprintf("%s->%s", callerFuncName, calleeFuncName))
}

func hashByMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func isExt(pkgPath string) bool {
	// TODO: gx/ipfs를 ext로 처리하기 위해선, path 자체에서 처리해주는 로직을 추가해야 함.
	return strings.Contains(pkgPath, "vendor") || strings.Contains(pkgPath, "Godeps/_workspace") || strings.Contains(pkgPath, "gx/ipfs")
}

func isStd(pkgPath string) bool {
	if strings.Contains(pkgPath, "golang.org") {
		return true
	}

	firstPath := strings.Split(pkgPath, "/")[0]
	return stdlib[firstPath]
}

func traverseSubDir(rootDir string) []string {
	max := 0
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if !strings.Contains(path, ".git") && !strings.Contains(path, "/vendor/") && !strings.Contains(path, "/Godeps/_workspace/") {
				max++
			}
		}
		return nil
	})

	count := 0
	allMains := make([]string, 0)
	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if !strings.Contains(path, ".git") && !strings.Contains(path, "/vendor") && !strings.Contains(path, "/Godeps/_workspace") {
				// _, file := filepath.Split(path)
				pkgName := strings.Join(strings.Split(path, "/")[6:], "/")
				// fmt.Printf("%s, %s\n", pkgName, file)

				program, error := buildProgram(pkgName)
				if error != nil {
					return nil
				}
				count++
				fmt.Printf("(%d/%d) %s done.\n", count, max, pkgName)

				for _, main := range getAllMains(program) {
					allMains = append(allMains, main)
				}
			}
		}
		return nil
	})

	return allMains
}

func getAllMains(program *ssa.Program) []string {
	pkgs := program.AllPackages()

	var mains []*ssa.Package
	mains = append(mains, ssautil.MainPackages(pkgs)...)

	if len(mains) == 0 {
		return nil
	}

	var mainPaths []string
	for _, main := range mains {
		mainPaths = append(mainPaths, main.Pkg.Path())
	}

	return mainPaths
}
