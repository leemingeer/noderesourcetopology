package utils

import "flag"

// KlogFlagVal is a wrapper to allow dynamic control of klog from the config file
type KlogFlagVal struct {
	flag             *flag.Flag
	isSetFromCmdLine bool
}
