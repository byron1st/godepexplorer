package extractor

import (
	"os"
	"path"
)

// Package info
type Package struct {
	ID          string          `json:"id"`
	Name        string          `json:"label"`
	PackagePath string          `json:"packagePath"`
	PackageName string          `json:"packageName"`
	PackageDir  string          `json:"packageDir"`
	IsPkg       bool            `json:"isPkg"`
	IsExternal  bool            `json:"isExternal"`
	IsStd       bool            `json:"isStd"`
	FuncSet     map[string]bool `json:"funcSet"`
}

// Dep is a struct to contain dependency relationship info
type Dep struct {
	ID        string          `json:"id"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Type      DepType         `json:"type"`
	Count     int             `json:"count"`
	DepAtFunc map[string]bool `json:"depAtFunc"`
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
	"syscall": true, "testing": true, "text": true, "time": true, "unicode": true, "unsafe": true,
}
