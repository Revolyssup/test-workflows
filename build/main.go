package main

import (
	"fmt"
)

var w *string
var force *bool
var version *string
var method *string
var url *string
var path *string

func main() {
	err := CreateComponents(StaticCompConfig{
		URL:     *url,
		Method:  *method,
		Path:    *path,
		DirName: *version,
		Config:  NewConfig(*version),
		Force:   *force,
	})
	if err != nil {
		fmt.Println("Failed to generate components: ", err.Error())
	}
}
