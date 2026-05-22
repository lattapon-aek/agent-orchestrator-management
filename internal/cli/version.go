package cli

import "fmt"

var Version = "dev"
var Commit = "unknown"
var BuiltAt = "unknown"
var GoVersion = "unknown"
var Dirty = "unknown"

func (r Runner) executeVersion() error {
	fmt.Fprintf(r.stdout, "aom version %s\n", Version)
	fmt.Fprintf(r.stdout, "commit %s\n", Commit)
	fmt.Fprintf(r.stdout, "built %s\n", BuiltAt)
	fmt.Fprintf(r.stdout, "go %s\n", GoVersion)
	fmt.Fprintf(r.stdout, "dirty %s\n", Dirty)
	return nil
}
