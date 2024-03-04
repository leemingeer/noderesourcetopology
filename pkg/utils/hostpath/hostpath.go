package hostpath

import "path/filepath"

var (
	pathPrefix = "/"
	SysfsDir   = HostDir(pathPrefix + "sys")
	VarDir     = HostDir(pathPrefix + "var")
	LibDir     = HostDir(pathPrefix + "lib")
)

type HostDir string

// Path returns a full path to a file under HostDir
func (d HostDir) Path(elem ...string) string {
	return filepath.Join(append([]string{string(d)}, elem...)...)
}
