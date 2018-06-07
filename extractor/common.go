package extractor

import (
	"os"
	"path"
)

// Pkg info
type Pkg struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Type  PkgType  `json:"type"`
	Meta  *PkgMeta `json:"meta"`
}

// PkgMeta is a meta info for a package object
type PkgMeta struct {
	PkgPath         string          `json:"pkgPath"`
	PkgName         string          `json:"pkgName"`
	PkgDir          string          `json:"pkgDir"`
	PkgType         PkgType         `json:"pkgType"`
	SinkEdgeIDSet   map[string]bool `json:"sinkEdgeIDSet"`
	SourceEdgeIDSet map[string]bool `json:"sourceEdgeIDSet"`
	Parent          string          `json:"parent"`
	Children        map[string]bool `json:"children"`
}

// PkgType is an enum for package type
type PkgType string

// NOR is a PkgType to denote the normal package.
// EXT is a PkgType to denote the external package.
// STD is a PkgType to denote the standard package.
const (
	NOR PkgType = "nor"
	EXT PkgType = "ext"
	STD PkgType = "std"
)

// Dep is a struct to contain dependency relationship info
type Dep struct {
	ID   string   `json:"id"`
	From string   `json:"from"`
	To   string   `json:"to"`
	Meta *DepMeta `json:"meta"`
}

// DepMeta is meta info for a dep object
type DepMeta struct {
	Type         DepType               `json:"type"`
	DepAtFuncSet map[string]*DepAtFunc `json:"depAtFuncSet"`
}

// DepAtFunc is a struct for dependencies at the function level
type DepAtFunc struct {
	ID   string `json:"id"`
	From *Func  `json:"from"`
	To   *Func  `json:"to"`
}

// Func is a struct for representing a function
type Func struct {
	Signature string `json:"signature"`
	Filename  string `json:"filename"`
}

// DepType is an enum for dependency relationship type
type DepType int

// COMP is a DepType to denote the composition relationship
// REL is a DepType to denote the normal use relationship
const (
	COMP DepType = iota
	REL
)

var gopath = path.Join(os.Getenv("GOPATH"), "src")
var stdlib = map[string]bool{
	"archive": true, "bufio": true, "builtin": true, "bytes": true, "compress": true, "container": true, "context": true, "crypto": true,
	"database": true, "debug": true, "encoding": true, "errors": true, "expvar": true, "flag": true, "fmt": true, "go": true, "hash": true,
	"html": true, "image": true, "index": true, "io": true, "log": true, "math": true, "mime": true, "net": true, "os": true, "path": true,
	"plugin": true, "reflect": true, "regexp": true, "runtime": true, "sort": true, "strconv": true, "strings": true, "sync": true,
	"syscall": true, "testing": true, "text": true, "time": true, "unicode": true, "unsafe": true, "internal": true,
}
