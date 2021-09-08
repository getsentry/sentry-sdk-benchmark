package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

// Compare compares runs from the same platform
func Compare(s []string) {

	for _, path := range s {
		info, err := os.Stat(path)
		if err != nil {
			panic(err)
		}

		if !info.Mode().IsDir() {
			panic(fmt.Errorf("[compare] %s is not a valid path", path))
		}

		files, err := ioutil.ReadDir(path)
		if err != nil {
			panic(err)
		}

		for _, f := range files {
			if f.IsDir() {
				name := f.Name()
				fmt.Println(name)
			}
		}
	}
}
