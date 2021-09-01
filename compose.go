package main

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
)

func Compose(benchmarkCfg BenchmarkConfig, runCfg RunConfig) (composeFile []byte, data DockerComposeData) {
	contextPath := path.Join(benchmarkCfg.Platform, runCfg.Name)
	dockerfile := findDockerfile(contextPath)
	resultPath := path.Join(append(
		strings.Split(benchmarkCfg.Platform, string(os.PathSeparator))[1:],
		fmt.Sprintf("%s-%s", benchmarkCfg.StartTime.Format("20060102-150405"), benchmarkCfg.ID),
		runCfg.Name,
	)...)
	data = DockerComposeData{
		ID:             benchmarkCfg.ID,
		RunName:        runCfg.Name,
		PlatformConfig: benchmarkCfg.PlatformConfig,
		App: App{
			ContextPath: contextPath,
			Dockerfile:  dockerfile,
		},
		ResultPath: resultPath,
		NeedsRelay: runCfg.NeedsRelay,
	}
	var b bytes.Buffer
	err := dockerComposeTemplate.Execute(&b, data)
	if err != nil {
		panic(err)
	}
	return b.Bytes(), data
}
