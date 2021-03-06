package extractor

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
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

var ErrNoSuchAlgorithm = errors.New("no such algorithm")
var ErrNoMainPackage = errors.New("there is no main package")

func GetDepsWithAlgorithm(rootPkgPath string, algorithm string) ([]*Pkg, []*Dep, error) {
	program, err := buildProgram(rootPkgPath)

	if err != nil {
		return nil, nil, err
	}

	var pkgSet map[string]*Pkg
	var depSet map[string]*Dep

	switch algorithm {
	case "static":
		pkgSet, depSet = inspectPackageWithStatic(program, rootPkgPath)
	case "cha":
		pkgSet, depSet = inspectPackageWithCHA(program, rootPkgPath)
	case "rta":
		pkgSet, depSet, err = inspectPackageWithRTA(program, rootPkgPath)
		if err != nil {
			return nil, nil, err
		}
	case "pointer":
		pkgSet, depSet, err = inspectPackageWithPointer(program, rootPkgPath)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, ErrNoSuchAlgorithm
	}

	if pkgSet == nil || depSet == nil {
		return nil, nil, ErrNoMainPackage
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

func buildProgram(rootPkgPath string) (*ssa.Program, error) {
	pkgPaths := []string{rootPkgPath}
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

func inspectPackageWithStatic(program *ssa.Program, rootPkgPath string) (map[string]*Pkg, map[string]*Dep) {
	fmt.Println("Analyze only static calls")

	return constructTree(traverseCallgraph(static.CallGraph(program), rootPkgPath))
}

func inspectPackageWithCHA(program *ssa.Program, rootPkgPath string) (map[string]*Pkg, map[string]*Dep) {
	fmt.Println("Analyze using the Class Hierarchy Analysis(CHA) algorithm")

	return constructTree(traverseCallgraph(cha.CallGraph(program), rootPkgPath))
}

func inspectPackageWithRTA(program *ssa.Program, rootPkgPath string) (map[string]*Pkg, map[string]*Dep, error) {
	fmt.Println("Analyze using the Rapid Type Analysis(RTA) algorithm")

	mains, err := getAllMains(program)
	if err != nil {
		return nil, nil, err
	}

	var roots []*ssa.Function
	for _, main := range mains {
		roots = append(roots, main.Func("init"), main.Func("main"))
	}
	cg := rta.Analyze(roots, true).CallGraph

	pkgs, deps := constructTree(traverseCallgraph(cg, rootPkgPath))
	return pkgs, deps, nil
}

func inspectPackageWithPointer(program *ssa.Program, rootPkgPath string) (map[string]*Pkg, map[string]*Dep, error) {
	fmt.Println("Analyze using the inclusion-based Points-To Analysis algorithm")

	mains, err := getAllMains(program)
	if err != nil {
		return nil, nil, err
	}

	config := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}

	analysis, err := pointer.Analyze(config)
	if err != nil {
		return nil, nil, err
	}

	pkgs, deps := constructTree(traverseCallgraph(analysis.CallGraph, rootPkgPath))
	return pkgs, deps, nil
}

func getAllMains(program *ssa.Program) ([]*ssa.Package, error) {
	allPkgs := program.AllPackages()

	var mains []*ssa.Package
	mains = append(mains, ssautil.MainPackages(allPkgs)...)

	if len(mains) == 0 {
		return nil, ErrNoMainPackage
	}

	return mains, nil
}

func traverseCallgraph(cg *callgraph.Graph, rootPkgPath string) (map[string]*Pkg, map[string]*Dep) {
	pkgSet := make(map[string]*Pkg)
	depSet := make(map[string]*Dep)

	callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		if isSynthetic(edge) {
			return nil
		}

		// Remove an edge if packages of its caller and callee are same
		if getPkgPath(edge.Caller) == getPkgPath(edge.Callee) {
			return nil
		}

		addPkg(pkgSet, edge.Caller, rootPkgPath)
		addPkg(pkgSet, edge.Callee, rootPkgPath)
		addDep(depSet, edge, pkgSet)

		return nil
	})

	return pkgSet, depSet
}

// TODO: should be fixed.
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

func addPkg(pkgSet map[string]*Pkg, node *callgraph.Node, rootPkgPath string) {
	pkgPath := getPkgPath(node)
	if pkgObj := pkgSet[getPkgIDFromPath(pkgPath)]; pkgObj == nil {
		newPkg := &Pkg{
			ID:    getPkgIDFromPath(pkgPath),
			Label: getPkgName(node),
			Type:  getPkgTypeFromPath(pkgPath, rootPkgPath),
			Meta: &PkgMeta{
				PkgPath:         pkgPath,
				PkgName:         getPkgName(node),
				PkgDir:          getPkgDirFromPath(pkgPath),
				PkgType:         getPkgTypeFromPath(pkgPath, rootPkgPath),
				SourceEdgeIDSet: make(map[string]bool),
				SinkEdgeIDSet:   make(map[string]bool),
				Parent:          "",
				Children:        make(map[string]bool),
			},
		}

		pkgSet[newPkg.ID] = newPkg
	}
}

func addDep(depSet map[string]*Dep, edge *callgraph.Edge, pkgSet map[string]*Pkg) {
	depID := getDepID(edge)
	depAtFunc := getDepAtFunc(edge)

	if depObj := depSet[depID]; depObj == nil {
		newDep := &Dep{
			ID:   depID,
			From: getPkgIDFromPath(getPkgPath(edge.Caller)),
			To:   getPkgIDFromPath(getPkgPath(edge.Callee)),
			Meta: &DepMeta{
				DepAtFuncSet: map[string]*DepAtFunc{depAtFunc.ID: depAtFunc},
				Type:         REL,
			},
		}

		depSet[depID] = newDep
	} else {
		depObj.Meta.DepAtFuncSet[depAtFunc.ID] = depAtFunc
	}

	if callerPkg := pkgSet[getPkgIDFromPath(getPkgPath(edge.Caller))]; callerPkg != nil {
		callerPkg.Meta.SourceEdgeIDSet[depID] = true
	}

	if calleePkg := pkgSet[getPkgIDFromPath(getPkgPath(edge.Callee))]; calleePkg != nil {
		calleePkg.Meta.SinkEdgeIDSet[depID] = true
	}
}

func getDepAtFunc(edge *callgraph.Edge) *DepAtFunc {
	return &DepAtFunc{
		ID:   getDepAtFuncID(edge),
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

func getPkgPath(node *callgraph.Node) string {
	return node.Func.Pkg.Pkg.Path()
}

func getPkgDirFromPath(pkgPath string) string {
	pkgDir := path.Join(gopath, pkgPath)

	if isStd(pkgPath) {
		return pkgPath
	}

	return pkgDir
}

func getPkgTypeFromPath(pkgPath string, rootPkgPath string) PkgType {
	if isStd(pkgPath) {
		return STD
	} else if isExt(pkgPath, rootPkgPath) {
		return EXT
	} else {
		return NOR
	}
}

func getFunc(node *callgraph.Node) *Func {
	funcName := node.Func.Name()
	funcSig := node.Func.Signature.String()[4:]
	fileName := node.Func.Prog.Fset.Position(node.Func.Pos()).Filename

	return &Func{
		Signature: fmt.Sprint(funcName, funcSig),
		Filename:  fileName,
	}
}

func getPkgIDFromPath(pkgPath string) string {
	return hashByMD5(pkgPath)
}

func getDepID(edge *callgraph.Edge) string {
	callerPkgID := getPkgIDFromPath(getPkgPath(edge.Caller))
	calleePkgID := getPkgIDFromPath(getPkgPath(edge.Callee))

	return hashByMD5(fmt.Sprintf("%s->%s", callerPkgID, calleePkgID))
}

func getCompDepID(parentPkgID string, childPkgID string) string {
	return hashByMD5(fmt.Sprintf("%s<>-%s", parentPkgID, childPkgID))
}

func getDepAtFuncID(edge *callgraph.Edge) string {
	callerFuncName := getFunc(edge.Caller).Signature
	calleeFuncName := getFunc(edge.Callee).Signature

	return hashByMD5(fmt.Sprintf("%s->%s", callerFuncName, calleeFuncName))
}

func hashByMD5(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	return hex.EncodeToString(hash.Sum(nil))
}

func isExt(pkgPath string, rootPkgPath string) bool {
	return !strings.HasPrefix(pkgPath, rootPkgPath)
}

func isStd(pkgPath string) bool {
	firstPath := strings.Split(pkgPath, "/")[0]
	return stdlib[firstPath]
}
