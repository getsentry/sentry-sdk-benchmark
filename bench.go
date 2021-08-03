package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var startTime = time.Now().UTC()

var dockerComposeTemplate *template.Template
var plotFilesTemplate *template.Template

func init() {
	dockerComposeTemplate = template.Must(template.ParseFiles("docker-compose.yml.tmpl"))
	plotFilesTemplate = template.Must(template.ParseFiles("template.html"))
}

type SummaryFile struct {
	Title            string
	BaselineHDR      string
	InstrumentedHDR  string
	BaselineJSON     string
	InstrumentedJSON string
}

type Benchmark struct {
	RunID       RunID
	App         App
	ResultPath  string
	DeployRelay bool
}

type App struct {
	ContainerName string
	ContextPath   string
	Dockerfile    string
	HostPort      int
	ContainerPort int
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

func bench(platform string) {
	id := NewRunID()

	log.Printf("START %s %s", id, platform)
	defer log.Printf("END   %s %s", id, platform)

	baseline := doRun(id, platform, "baseline", false)
	instrumented := doRun(id, platform, "instrumented", true)

	compare(baseline, instrumented)
}

type RunResult struct {
	ComposeFile []byte
	Path        string
}

func doRun(id RunID, platform, label string, deployRelay bool) *RunResult {
	containerName := filepath.Base(platform)
	// projectName := fmt.Sprintf("%s-%s-%s", containerName, label, id)
	contextPath := path.Join(platform, label)
	resultPath := path.Join(append(
		strings.Split(platform, string(os.PathSeparator))[1:],
		fmt.Sprintf("%s-%s", startTime.Format("20060102-150405"), id),
		label,
	)...)
	dockerfile := findDockerfile(contextPath)

	var b bytes.Buffer
	err := dockerComposeTemplate.Execute(&b, Benchmark{
		RunID: id,
		App: App{
			ContainerName: containerName,
			ContextPath:   contextPath,
			Dockerfile:    dockerfile,
		},
		ResultPath:  resultPath,
		DeployRelay: deployRelay,
	})
	if err != nil {
		panic(err)
	}

	// setUp(projectName, b.Bytes())
	// defer tearDown(projectName)

	// waitUntilExit("loadgen-" + id.String())

	result := &RunResult{
		ComposeFile: b.Bytes(),
		Path:        filepath.Join("result", filepath.Join(strings.Split(resultPath, "/")...)),
	}

	// if err := os.MkdirAll(result.Path, 0777); err != nil {
	// 	panic(err)
	// }
	// if err := os.WriteFile(filepath.Join(result.Path, "docker-compose.yml"), result.ComposeFile, 0666); err != nil {
	// 	panic(err)
	// }

	return result
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

func setUp(projectName string, composeFile []byte) {
	cmd := exec.Command(
		"docker", "compose",
		"--project-name", projectName,
		"--file", "-",
		"up", "--detach",
	)
	cmd.Stdin = bytes.NewReader(composeFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func tearDown(projectName string) {
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

func waitUntilExit(containerName string) {
	log.Printf("Waiting for container %s to stop", containerName)
	b, err := exec.Command("docker", "wait", containerName).CombinedOutput()
	if err != nil {
		panic(err)
	}

	if status := string(bytes.TrimSpace(b)); status != "0" {
		panic("Container exited with status " + status)
	}
}

func read(path string) string {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	return string(file)
}

func generateSummary(path string) {
	folderName := filepath.Dir(path)
	// summaryPath = filepath.Join(folderPath, "summary.html")
	betterPath := "result/python/django/20210803-190205-ujefeaq"
	var b bytes.Buffer
	err := plotFilesTemplate.Execute(&b, SummaryFile{
		Title:            folderName,
		BaselineHDR:      read(filepath.Join(betterPath, "baseline.hdr")),
		InstrumentedHDR:  read(filepath.Join(betterPath, "instrumented.hdr")),
		BaselineJSON:     read(filepath.Join(betterPath, "baseline.json")),
		InstrumentedJSON: read(filepath.Join(betterPath, "instrumented.json")),
	})
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(betterPath, "summary.html"), b.Bytes(), 0666); err != nil {
		panic(err)
	}
	// baseline.hdr
	// instrumented.json
	// instrumented.hdr
	// instrumented.hdr
}

func compare(before, after *RunResult) {
	// TODO
	fmt.Println(before.Path)
	generateSummary(before.Path)
}
