// Wrappers around the io calls which perform some rudimentary
// error reporting and then abort the program.
package main

import (
	"log"
	"mm"
)

func Create(name string, size int, p interface{}) {
	err := mm.Create(name, size, p)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Allocate(size int, p interface{}) {
	err := mm.Allocate(size, p)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Close(p interface{}) {
	err := mm.Close(p)
	if err != nil {
		log.Fatal(err.Error())
	}
}
