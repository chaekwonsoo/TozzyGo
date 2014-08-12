package main

import (
	"runtime"
	"log"
)

func main() {
	log.SetFlags(log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())

	tozzy()
}
