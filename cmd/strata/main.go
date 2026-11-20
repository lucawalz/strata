package main

import (
	"fmt"
	"os"
)

const exitUsage = 2

func main() {
	fmt.Fprintln(os.Stderr, "usage: strata check [--format text|github] [path]")
	os.Exit(exitUsage)
}
