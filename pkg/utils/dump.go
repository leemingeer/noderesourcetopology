package utils

import (
	"fmt"
	"sigs.k8s.io/yaml"
)

type dumper struct {
	obj interface{}
}

// String implements the fmt.Stringer interface
func (d *dumper) String() string {
	return Dump(d.obj)
}

// Dump dumps an object into YAML textual format
func Dump(obj interface{}) string {
	out, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("<!!! FAILED TO MARSHAL %T (%v) !!!>\n", obj, err)
	}
	return string(out)
}

// DelayedDumper delays the dumping of an object. Useful in logging to delay
// the processing (JSON marshalling) until (or if) the object is actually
// evaluated.
func DelayedDumper(obj interface{}) fmt.Stringer {
	return &dumper{obj: obj}
}
