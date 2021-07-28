package main

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/schema"
)

type Benchmark struct {
	Name   string
	Suffix string
	App    App
}

type App struct {
	ContainerName  string
	ContextPath    string
	Dockerfile     string
	HostPort       int
	ContainerPort  int
	IsInstrumented bool
}

func validateComposeFile(b []byte) error {
	m, err := loader.ParseYAML(b)
	if err != nil {
		return err
	}
	return schema.Validate(m)
}

func main() {
	tmpl, err := template.ParseFiles("docker-compose.yml.tmpl")
	if err != nil {
		panic(err)
	}
	for _, instrumented := range []bool{true, false} {
		var b bytes.Buffer
		// TODO: get benchmark config from benchmark_config.json
		err = tmpl.Execute(&b, Benchmark{
			Name:   "django-instrumented",
			Suffix: "aeoijfs0",
			App: App{
				ContainerName:  "django",
				ContextPath:    "platform/python/django/instrumented",
				Dockerfile:     "django-postgres.dockerfile",
				HostPort:       8080,
				ContainerPort:  8080,
				IsInstrumented: instrumented,
			},
		})
		if err != nil {
			panic(err)
		}
		fmt.Println(b.String())
		if err := validateComposeFile(b.Bytes()); err != nil {
			panic(err)
		}
	}
}
