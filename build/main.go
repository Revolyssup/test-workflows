package main

import (
	"flag"
	"fmt"
)

func main() {

	force := flag.Bool("force", false, "")
	version := flag.String("version", LatestVersion, "")
	path := flag.String("path", workloadPath, "")
	method := flag.String("method", DefaultGenerationMethod, "")
	url := flag.String("url", DefaultGenerationURL, "")
	flag.Parse()

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
