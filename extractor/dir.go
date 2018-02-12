package extractor

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
)

// GetDirTree extracts a list of packages and composition relationships based on the directory structure.
func GetDirTree(rootPkgName string) ([]*Package, []*Dep, error) {
	rootPkgPathStr := path.Join(gopath, rootPkgName)
	fmt.Printf("Root package path: %s\n", rootPkgPathStr)
	packageList, depList, err := traverse(rootPkgPathStr)
	if err != nil {
		return nil, nil, err
	}

	return packageList, depList, nil
}

func traverse(pathStr string) ([]*Package, []*Dep, error) {
	packageList := make([]*Package, 0)
	depList := make([]*Dep, 0)
	goPathLen := len(gopath) + 1
	isPkg := false

	files, err := ioutil.ReadDir(pathStr)
	if err != nil {
		return nil, nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".go" {
			isPkg = true
		}

		childPathStr := path.Join(pathStr, file.Name())

		if file.IsDir() {
			fmt.Printf("%s - %s\n", pathStr, childPathStr)
			childPackageList, childDepList, err := traverse(childPathStr)

			if err != nil {
				return nil, nil, err
			}

			childPathStr := childPackageList[len(childPackageList)-1].ID

			depList = append(depList, &Dep{
				ID:    fmt.Sprintf("%s<>-%s", pathStr, childPathStr),
				From:  childPathStr,
				To:    pathStr,
				Type:  COMP,
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
		ID:          pathStr,
		Name:        name,
		PackagePath: pathStr[goPathLen:],
		PackageDir:  pathStr,
		IsPkg:       isPkg,
		FuncSet:     make(map[string]bool),
	})

	return packageList, depList, nil
}
