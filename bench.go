package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	exec "github.com/getsentry/sentry-sdk-benchmark/internal/std/execabs"
)

var dockerComposeTemplate = template.Must(template.ParseFiles(filepath.Join("template", "docker-compose.yml.tmpl")))

type BenchmarkConfig struct {
	ID        BenchmarkID
	StartTime time.Time
	Platform  string // a valid path like platform/python/django
	Runs      []RunConfig
}

func BenchmarkConfigFromPlatform(platform string) BenchmarkConfig {
	cfg := BenchmarkConfig{
		ID:        NewBenchmarkID(),
		StartTime: time.Now().UTC(),
	}
	switch filepath.Base(platform) {
	case "baseline":
		cfg.Platform = filepath.Dir(platform)
		cfg.Runs = []RunConfig{
			{
				Name:       "baseline",
				NeedsRelay: false,
			},
		}
	case "instrumented":
		cfg.Platform = filepath.Dir(platform)
		cfg.Runs = []RunConfig{
			{
				Name:       "instrumented",
				NeedsRelay: true,
			},
		}
	default:
		cfg.Platform = platform
		cfg.Runs = []RunConfig{
			{
				Name:       "baseline",
				NeedsRelay: false,
			},
			{
				Name:       "instrumented",
				NeedsRelay: true,
			},
		}
	}
	return cfg
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

func Benchmark(ctx context.Context, cfg BenchmarkConfig) {
	oldprefix := log.Prefix()
	defer log.SetPrefix(oldprefix)
	log.SetPrefix(fmt.Sprintf("%s[%s] ", oldprefix, cfg.ID))

	var results []*RunResult
	for _, runCfg := range cfg.Runs {
		results = append(results, run(ctx, cfg, runCfg))
	}

	report(results)
}

type RunResult struct {
	Name        string
	ComposeFile []byte
	Path        string
}

func run(ctx context.Context, benchmarkCfg BenchmarkConfig, runCfg RunConfig) *RunResult {
	oldprefix := log.Prefix()
	defer log.SetPrefix(oldprefix)
	log.SetPrefix(fmt.Sprintf("%s[%s] ", oldprefix, path.Join(append(strings.Split(benchmarkCfg.Platform, string(os.PathSeparator))[1:], runCfg.Name)...)))

	log.Print("START")
	defer log.Print("END")

	projectName := fmt.Sprintf("%s-%s-%s-%s",
		filepath.Base(filepath.Dir(benchmarkCfg.Platform)),
		filepath.Base(benchmarkCfg.Platform),
		runCfg.Name,
		benchmarkCfg.ID)
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
			ContextPath: contextPath,
			Dockerfile:  dockerfile,
		},
		ResultPath: resultPath,
		NeedsRelay: runCfg.NeedsRelay,
	})
	if err != nil {
		panic(err)
	}

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

	defer composeDown(projectName)
	composeBuild(ctx, projectName, result.ComposeFile)
	composeUp(ctx, projectName, result.ComposeFile, filepath.Join(result.Path, "docker-compose-up.log"))

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

func composeBuild(ctx context.Context, projectName string, composeFile []byte) {
	log.Print("Running 'docker compose build'...")
	cmd := exec.CommandContext(
		ctx,
		"docker", "compose",
		"--project-name", projectName,
		"--file", "-",
		"build",
	)
	cmd.Stdin = bytes.NewReader(composeFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func composeUp(ctx context.Context, projectName string, composeFile []byte, outpath string) {
	log.Printf("Running 'docker compose up', streaming logs to %q...", outpath)

	out, err := os.Create(outpath)
	if err != nil {
		panic(err)
	}
	defer out.Close()

	cmd := exec.CommandContext(
		ctx,
		"docker", "compose",
		"--project-name", projectName,
		"--file", "-",
		"up", "--exit-code-from", "loadgen",
	)
	cmd.Stdin = bytes.NewReader(composeFile)
	cmd.Stdout = io.MultiWriter(out, os.Stdout)
	cmd.Stderr = io.MultiWriter(out, os.Stderr)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func composeDown(projectName string) {
	log.Print("Running 'docker compose down'...")
	cmd := exec.Command(
		"docker", "compose",
		"--project-name", projectName,
		"down", "--remove-orphans", "--rmi", "local")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
