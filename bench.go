package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
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
	ID             BenchmarkID
	StartTime      time.Time
	Platform       string         // a valid path like platform/python/django
	PlatformConfig PlatformConfig // from platform/*/*/config.json
	Runs           []RunConfig
}

type PlatformConfig struct {
	Target struct {
		Path string
	}
	RPS      uint16
	Duration string
}

func (cfg PlatformConfig) Validate() error {
	if cfg.Target.Path == "" {
		return fmt.Errorf(`platform config missing "target.path"`)
	}
	if cfg.RPS == 0 {
		return fmt.Errorf(`platform config missing "rps"`)
	}
	d, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		return fmt.Errorf(`platform config invalid "duration": %q: %s`, cfg.Duration, err)
	}
	if d <= 0 {
		return fmt.Errorf(`platform config nonpositive "duration": %q`, cfg.Duration)
	}
	return nil
}

// BenchmarkConfigFromPath returns the necessary configuration to run a
// benchmark targeting the app or apps at the given path.
//
// Path must be either a directory with a configuration file and one or more app
// directories, or path must be an app directory whose parent contains a
// configuration file.
//
// BenchmarkConfigFromPath panics if it cannot create a valid configuration from
// the given path.
//
// Path is always cleaned with filepath.Clean, such that equivalent spellings of
// the same path will return equivalent configuration.
func BenchmarkConfigFromPath(path string) BenchmarkConfig {
	path = filepath.Clean(path)
	if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
		panic(fmt.Errorf("could not read directory: %q", path))
	}
	pcpath := MustFindPlatformConfig(path)
	cfg := BenchmarkConfig{
		ID:             NewBenchmarkID(),
		StartTime:      time.Now().UTC(),
		Platform:       filepath.Dir(pcpath),
		PlatformConfig: MustReadPlatformConfig(pcpath),
	}
	var apps []string
	if cfg.Platform == path {
		// all apps of the platform
		apps = subDirs(path)
	} else {
		// single app
		apps = []string{path}
	}
	if len(apps) == 0 {
		panic(fmt.Errorf("no app to benchmark in %q", path))
	}
	for _, app := range apps {
		name := filepath.Base(app)
		cfg.Runs = append(cfg.Runs, RunConfig{
			Name:       name,
			NeedsRelay: name != "baseline",
		})
	}
	return cfg
}

// MustFindPlatformConfig returns the path to the platform configuration for the
// given path. Path itself must contain a configuration file or path's parent
// directory must contain a configuration file. MustFindPlatformConfig panics if
// a configuration file cannot be found.
func MustFindPlatformConfig(path string) string {
	candidates := []string{
		filepath.Join(path, "config.json"),
		filepath.Join(filepath.Dir(path), "config.json"),
	}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	panic(fmt.Errorf("no config file found in: %q", candidates))
}

// MustReadPlatformConfig reads and validates a PlatformConfig from path. It
// panics if the configuration cannot be read or is invalid.
func MustReadPlatformConfig(path string) PlatformConfig {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var pc PlatformConfig
	err = json.NewDecoder(f).Decode(&pc)
	if err != nil {
		panic(err)
	}
	if err := pc.Validate(); err != nil {
		panic(err)
	}
	return pc
}

// subDirs returns all subdirectories of path.
func subDirs(path string) []string {
	var s []string
	entries, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s = append(s, e.Name())
	}
	return s
}

type RunConfig struct {
	Name       string
	NeedsRelay bool
}

type DockerComposeData struct {
	ID             BenchmarkID
	RunName        string
	PlatformConfig PlatformConfig
	App            App
	ResultPath     string
	NeedsRelay     bool
	Language       string
	Framework      string
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

	language := filepath.Base(filepath.Dir(benchmarkCfg.Platform))
	framework := filepath.Base(benchmarkCfg.Platform)

	projectName := fmt.Sprintf("%s-%s-%s-%s", language, framework, runCfg.Name, benchmarkCfg.ID)
	contextPath := path.Join(benchmarkCfg.Platform, runCfg.Name)
	resultPath := path.Join(append(
		strings.Split(benchmarkCfg.Platform, string(os.PathSeparator))[1:],
		fmt.Sprintf("%s-%s", benchmarkCfg.StartTime.Format("20060102-150405"), benchmarkCfg.ID),
		runCfg.Name,
	)...)
	dockerfile := findDockerfile(contextPath)

	var b bytes.Buffer
	err := dockerComposeTemplate.Execute(&b, DockerComposeData{
		ID:             benchmarkCfg.ID,
		RunName:        runCfg.Name,
		PlatformConfig: benchmarkCfg.PlatformConfig,
		App: App{
			ContextPath: contextPath,
			Dockerfile:  dockerfile,
		},
		ResultPath: resultPath,
		NeedsRelay: runCfg.NeedsRelay,
		Language:   language,
		Framework:  framework,
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
