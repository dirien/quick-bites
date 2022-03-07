package main

import (
	"fmt"
)

var (
	version = "0.0.1"
	commit  = "none"
	date    = "none"
	builtBy = "none"
)

func main() {
	fmt.Println("Version:\t", version)
	fmt.Println("Commit:\t\t", commit)
	fmt.Println("Date:\t\t", date)
	fmt.Println("Built by:\t", builtBy)
}
