//+build !test

package main

import (
	"log"
	"os"
	"fmt"
)

const version = "0.1.0"

func main() {
	var svc Service

	fmt.Println("Service init...")
	err := svc.Init()
	if err != nil {
		log.Fatal(err.Error())
	}

	if err != nil {
		os.Exit(-1)
	}
}