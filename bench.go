package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var startTime = time.Now().UTC()

var dockerComposeTemplate *template.Template

func init() {
	var err error
	dockerComposeTemplate, err = template.ParseFiles("docker-compose.yml.tmpl")
	if err != nil {
		panic(err)
	}
}

type Benchmark struct {
	RunID      RunID
	App        App
	ResultPath string
}

type App struct {
	ContainerName  string
	ContextPath    string
	Dockerfile     string
	HostPort       int
	ContainerPort  int
	IsInstrumented bool
}

type RunID [4]byte

func NewRunID() RunID {
	var id RunID
	_, err := rand.Read(id[:])
	if err != nil {
		panic(err)
	}
	return id
}

var base32Encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func (r RunID) String() string {
	return base32Encoding.EncodeToString(r[:])
}

type Result interface{}

func bench(platform string) {
	id := NewRunID()
	log.Printf("START %s %s", id, platform)
	defer log.Printf("END   %s %s", id, platform)

	baselineResult := doRun(id, platform, false)
	instrumentedResult := doRun(id, platform, true)

	compare(baselineResult, instrumentedResult)
}

func doRun(id RunID, platform string, instrumented bool) Result {
	containerName := filepath.Base(platform)
	var projectName, contextPath, resultPath string
	if instrumented {
		projectName = fmt.Sprintf("%s-%s-%s", containerName, "instrumented", id)
		contextPath = path.Join(platform, "instrumented")
		resultPath = path.Join(append(
			filepath.SplitList(platform)[1:],
			fmt.Sprintf("%s-%s", startTime.Format("20060102-150405"), id),
			"instrumented",
		)...)
	} else {
		projectName = fmt.Sprintf("%s-%s-%s", containerName, "baseline", id)
		contextPath = path.Join(platform, "baseline")
		resultPath = path.Join(append(
			filepath.SplitList(platform)[1:],
			fmt.Sprintf("%s-%s", startTime.Format("20060102-150405"), id),
			"baseline",
		)...)
	}
	dockerfile := findDockerfile(contextPath)

	var b bytes.Buffer
	err := dockerComposeTemplate.Execute(&b, Benchmark{
		RunID: id,
		App: App{
			ContainerName:  containerName,
			ContextPath:    contextPath,
			Dockerfile:     dockerfile,
			IsInstrumented: instrumented,
		},
		ResultPath: resultPath,
	})
	if err != nil {
		panic(err)
	}

	setUp := func() {
		cmd := exec.Command(
			"docker", "compose",
			"--project-name", projectName,
			"--file", "-",
			"up", "--detach",
		)
		cmd.Stdin = bytes.NewReader(b.Bytes())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
	tearDown := func() {
		cmd := exec.Command(
			"docker", "compose",
			"--project-name", projectName,
			"down", "--remove-orphans")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}

	setUp()
	defer tearDown()

	waitUntilResultOrInterrupt(filepath.Join("result", filepath.Join(strings.Split(resultPath, "/")...)))

	return nil
}

func findDockerfile(path string) string {
	s, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, entry := range s {
		if !entry.IsDir() && strings.Contains(strings.ToLower(entry.Name()), "dockerfile") {
			return entry.Name()
		}
	}
	panic(fmt.Errorf("No Dockerfile in %s", path))
}

func waitUntilResultOrInterrupt(path string) {
	done := make(chan bool)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	log.Printf("Waiting for result at %s, Ctrl-C to stop waiting", path)

	go func() {
		for {
			_, err := os.Stat(path)
			if errors.Is(err, os.ErrNotExist) {
				time.Sleep(time.Second)
				continue
			}
			if err == nil {
				close(done)
				return
			}
			panic(err)
		}
	}()

	select {
	case <-done:
	case <-ch:
	}
}

func compare(baseline, instrumented Result) {}
