package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	flag.Parse()

	dir1 := flag.Arg(0)
	dir2 := flag.Arg(1)
	d, err := DiffDirs(dir1, dir2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(d)
}
