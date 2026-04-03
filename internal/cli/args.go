package cli

import "os"

// rawArgs returns os.Args[1:] — all arguments after the binary name.
// Extracted into a function for testability.
func rawArgs() []string {
	if len(os.Args) < 2 {
		return nil
	}
	return os.Args[1:]
}
