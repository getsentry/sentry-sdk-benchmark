package main

import (
	"log"
	"time"

	cadvisor "github.com/google/cadvisor/client/v2"
	cadvisor_info "github.com/google/cadvisor/info/v2"
)

type Stats struct {
	Before     ContainerStats           `json:"before"`
	After      ContainerStats           `json:"after"`
	Difference ContainerStatsDifference `json:"difference"`
}

type ContainerStats struct {
	Timestamp           time.Time `json:"timestamp"`
	MemoryMaxUsageBytes uint64    `json:"memory_max_usage_bytes"`
	CPUUsageUser        uint64    `json:"cpu_usage_user"`
	CPUUsageSystem      uint64    `json:"cpu_usage_system"`
	CPUUsageTotal       uint64    `json:"cpu_usage_total"`
}

type ContainerStatsDifference struct {
	Duration            time.Duration `json:"duration"`
	MemoryMaxUsageBytes int64         `json:"memory_max_usage_bytes"`
	CPUUsageUser        int64         `json:"cpu_usage_user"`
	CPUUsageSystem      int64         `json:"cpu_usage_system"`
	CPUUsageTotal       int64         `json:"cpu_usage_total"`
}

func containerStats(cAdvisorURL string, containerName string) ContainerStats {
	log.Printf("Fetching stats for container %q from %q", containerName, cAdvisorURL)
	client, err := cadvisor.NewClient(cAdvisorURL)
	if err != nil {
		panic(err)
	}
	opts := &cadvisor_info.RequestOptions{
		IdType: cadvisor_info.TypeDocker,
		Count:  1,
	}
	m, err := client.Stats(containerName, opts)
	if err != nil {
		panic(err)
	}
	for _, v := range m {
		return ContainerStats{
			Timestamp:           v.Stats[0].Timestamp,
			MemoryMaxUsageBytes: v.Stats[0].Memory.MaxUsage,
			CPUUsageUser:        v.Stats[0].Cpu.Usage.User,
			CPUUsageSystem:      v.Stats[0].Cpu.Usage.System,
			CPUUsageTotal:       v.Stats[0].Cpu.Usage.Total,
		}
	}
	panic("missing cAdvisor stats")
}
