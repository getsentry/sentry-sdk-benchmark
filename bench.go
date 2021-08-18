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

var dockerComposeTemplate = template.Must(template.ParseFiles(filepath.Join("template", "docker-compose.yml.tmpl")))

type BenchmarkConfig struct {
	ID        BenchmarkID
	StartTime time.Time
	Platform  string // a valid path like platform/python/django
	Runs      []RunConfig
}

func BenchmarkConfigFromPlatform(platform string) BenchmarkConfig {
	return BenchmarkConfig{
		ID:        NewBenchmarkID(),
		StartTime: time.Now().UTC(),
		Platform:  platform,
		Runs: []RunConfig{
			{
				Name:       "baseline",
				NeedsRelay: false,
			},
			{
				Name:       "instrumented",
				NeedsRelay: true,
			},
		},
	}
}

type RunConfig struct {
	Name       string
	NeedsRelay bool
}

type DockerComposeData struct {
	ID         BenchmarkID
	RunName    string
	App        App
	ResultPath string
	NeedsRelay bool
}

type App struct {
	ContainerName string
	ContextPath   string
	Dockerfile    string
	HostPort      int
	ContainerPort int
}

type BenchmarkID [4]byte

func NewBenchmarkID() BenchmarkID {
	var id BenchmarkID
	_, err := rand.Read(id[:])
	if err != nil {
		panic(err)
	}
	return id
}

var base32Encoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

func (r BenchmarkID) String() string {
	return base32Encoding.EncodeToString(r[:])
}

func Benchmark(cfg BenchmarkConfig) {
	log.Printf("START %s %s", cfg.ID, cfg.Platform)
	defer log.Printf("END   %s %s", cfg.ID, cfg.Platform)

	var results []*RunResult
	for _, runCfg := range cfg.Runs {
		results = append(results, run(cfg, runCfg))
	}

	report(results)
}

type RunResult struct {
	Name        string
	ComposeFile []byte
	Path        string
}

func run(benchmarkCfg BenchmarkConfig, runCfg RunConfig) *RunResult {
	containerName := filepath.Base(benchmarkCfg.Platform)
	projectName := fmt.Sprintf("%s-%s-%s", containerName, runCfg.Name, benchmarkCfg.ID)
	contextPath := path.Join(benchmarkCfg.Platform, runCfg.Name)
	resultPath := path.Join(append(
		strings.Split(benchmarkCfg.Platform, string(os.PathSeparator))[1:],
		fmt.Sprintf("%s-%s", benchmarkCfg.StartTime.Format("20060102-150405"), benchmarkCfg.ID),
		runCfg.Name,
	)...)
	dockerfile := findDockerfile(contextPath)

	var b bytes.Buffer
	err := dockerComposeTemplate.Execute(&b, DockerComposeData{
		ID:      benchmarkCfg.ID,
		RunName: runCfg.Name,
		App: App{
			ContainerName: containerName,
			ContextPath:   contextPath,
			Dockerfile:    dockerfile,
		},
		ResultPath: resultPath,
		NeedsRelay: runCfg.NeedsRelay,
	})
	if err != nil {
		panic(err)
	}

	setUp(projectName, b.Bytes())
	defer tearDown(projectName)

	waitUntilExit("loadgen-" + runCfg.Name + "-" + benchmarkCfg.ID.String())

	result := &RunResult{
		Name:        runCfg.Name,
		ComposeFile: b.Bytes(),
		Path:        filepath.Join("result", filepath.Join(strings.Split(resultPath, "/")...)),
	}

	if err := os.MkdirAll(result.Path, 0777); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(result.Path, "docker-compose.yml"), result.ComposeFile, 0666); err != nil {
		panic(err)
	}

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
	panic(fmt.Errorf("no Dockerfile in %s", path))
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
		panic(fmt.Errorf("Container %s exited with status %s", containerName, status))
	}
}
