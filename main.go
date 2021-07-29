package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/schema"
	vegeta "github.com/tsenart/vegeta/v12/lib"
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

func parseTemplateYAML(b bytes.Buffer) (map[string]interface{}, error) {
	m, err := loader.ParseYAML(b.Bytes())
	if err != nil {
		return nil, err
	}

	err = schema.Validate(m)
	if err != nil {
		return nil, fmt.Errorf("could not validate yaml: \n%s", b.String())
	}

	return m, nil
}

func generateRandString(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func getPlatformPath() (string, error) {
	args := os.Args[1:]
	if len(args) != 1 {
		return "", errors.New("need to provide path to platform")
	}

	platformPath := args[0]
	if !strings.HasPrefix(platformPath, "platform/") {
		return "", fmt.Errorf("invalid platform path: %s", platformPath)
	}

	files, err := ioutil.ReadDir(platformPath)
	if err != nil {
		return "", fmt.Errorf("could not read directory of platform path: %s", platformPath)
	}

	// todo(abhi): have a stronger check here?
	if len(files) != 3 {
		return "", fmt.Errorf("path does not have the correct directory structure: %s", platformPath)
	}

	return platformPath, nil
}

// TODO: This should be configurable
func warmUpAttack() {
	rate := vegeta.Rate{Freq: 500, Per: time.Second}
	duration := 10 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8080/update?query=10",
	})
	attacker := vegeta.NewAttacker()

	for range attacker.Attack(targeter, rate, duration, "") {
	}
}

// TODO: This should be configurable as well, probably should just be 1 func
// TODO: Generate URL programatically
func finalAttack() vegeta.Reporter {
	rate := vegeta.Rate{Freq: 500, Per: time.Second}
	duration := 10 * time.Second
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: "GET",
		URL:    "http://localhost:8080/update?query=10",
	})
	attacker := vegeta.NewAttacker()

	var m vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "") {
		m.Add(res)
	}
	m.Close()

	return vegeta.NewHDRHistogramPlotReporter(&m)
}

type benchmarkConfig struct {
	Framework string `json:"framework"`
	Tests     []struct {
		Postgres map[string]interface{} `json:"postgresql"`
	} `json:"tests"`
}

func getBenchmarkConfig(platformPath string) (map[string]interface{}, error) {
	benchmarkConfigFile, err := os.Open(platformPath + "/common/benchmark_config.json")
	if err != nil {
		return nil, err
	}
	defer benchmarkConfigFile.Close()

	benchmarkConfigBytes, err := ioutil.ReadAll(benchmarkConfigFile)
	if err != nil {
		return nil, err
	}

	var config benchmarkConfig
	err = json.Unmarshal(benchmarkConfigBytes, &config)
	if err != nil {
		return nil, err
	}

	return config.Tests[0].Postgres, nil
}

func main() {
	tmpl, err := template.ParseFiles("docker-compose.yml.tmpl")
	if err != nil {
		panic(err)
	}

	platformPath, err := getPlatformPath()
	if err != nil {
		panic(err)
	}

	config, err := getBenchmarkConfig(platformPath)
	if err != nil {
		panic(err)
	}

	suffix, err := generateRandString(6)
	if err != nil {
		panic(err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	port := int(config["port"].(float64))

	for _, instrumented := range []bool{true, false} {
		name := path.Base(platformPath)
		contextPath := platformPath

		if instrumented {
			name += "-instrumented"
			contextPath += "/instrumented"
		} else {
			name += "-baseline"
			contextPath += "/baseline"
		}

		var b bytes.Buffer
		err = tmpl.Execute(&b, Benchmark{
			Name:   name,
			Suffix: suffix,
			App: App{
				ContainerName:  name,
				ContextPath:    contextPath,
				Dockerfile:     "django-postgresql.dockerfile",
				HostPort:       port,
				ContainerPort:  port,
				IsInstrumented: instrumented,
			},
		})
		if err != nil {
			panic(err)
		}

		fmt.Println(b.String())

		_, err := parseTemplateYAML(b)
		if err != nil {
			panic(err)
		}

		tmpComposeFile, err := os.CreateTemp(currentDir, "docker-compose-tmp-*.yml")
		if err != nil {
			panic(err)
		}
		defer os.Remove(tmpComposeFile.Name())

		_, err = b.WriteTo(tmpComposeFile)
		if err != nil {
			panic(err)
		}

		dockerComposeUpCmd := exec.Command("docker", "compose", "-f", path.Base(tmpComposeFile.Name()), "up", "--build")
		if err := dockerComposeUpCmd.Start(); err != nil {
			panic(err)
		}

		// TODO: have a better way to wait until docker compose has started
		time.Sleep(5000 * time.Millisecond)

		fmt.Println("Starting Warm up runs")
		for i := 0; i < 3; i++ {
			warmUpAttack()
		}

		fmt.Println("Start final attack")
		reporter := finalAttack()

		// Todo(abhi): Better file naming structure
		reportFile, err := os.OpenFile(name+"100", os.O_WRONLY|os.O_CREATE, 0777)
		if err != nil {
			panic(err)
		}

		defer reportFile.Close()
		reporter.Report(reportFile)

		// TODO: Figure out how to get this to work!!
		dockerComposeDownCmd := exec.Command("docker", "compose", "down")
		if err := dockerComposeDownCmd.Run(); err != nil {
			panic(err)
		}
	}
}
