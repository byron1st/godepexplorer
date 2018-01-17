package extractor

type Package struct {
	Id          string
	Name        string
	PackagePath string
	PackageName string
	PackageDir  string
	IsExternal  bool
	IsStd       bool
}

type Dep struct {
	Id    string
	From  string
	To    string
	Count int
}

func GetDirTree(rootPath string) (error, []*Package, []*Dep) {
	packageList := make([]*Package, 0)
	depList := make([]*Dep, 0)

	return nil, packageList, depList
}