package version

const undefinedVersion string = "undefined"

// Must not be const, supposed to be set using ldflags at build time
var version = undefinedVersion

// Get returns the version as a string
func Get() string {
	return version
}

// Undefined returns if version is at it's default value
func Undefined() bool {
	return version == undefinedVersion
}
