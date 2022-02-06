package main

import (
	"log"
	"github.com/tjgq/sane"
)

func main() {

	log.Printf("ScanServer v1.0.0")

	if err1 := sane.Init(); err1 != nil {
		panic(err1)
	}
	defer sane.Exit()

}
