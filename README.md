# Note

This repo is now archived. Please use https://github.com/getsentry/sdk-measurements/ for all SDK measurements going forward.

# Sentry SDK Benchmark

This repository contains reproducible benchmarks to evaluate the instrumentation overhead of Sentry SDKs.

The focus is on Performance Monitoring (tracing) of web servers.

## How It Works

```
                      ┌────────────────────┐
                      │┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼│
   ┌───────────┐      │┼┼────────────────┼┼┼┐      ┌────────┐
   │           │      │┼│                │┼┼│      │        │
   │ Benchmark ├─────►├┼│ Docker Compose │┼┼┼─────►│  HTML  │
   │  Runner   │      │┼│    Projects    │┼┼│      │ Report │
   │           │      │┼│                │┼┼│      │        │
   └──────────┬┘      │┼┼────────────────┼┼┼│      └────────┘
              │       └┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼┼│
              │        └────────────────────┘
              │
       ┌──────┴────────────────────────┐
       │ Selection of prepackaged apps │
       │   instrumented with Sentry    │
       └───────────────────────────────┘
```

We provide a [collection of prepackaged web apps](platform/) implemented in different programming languages and using different frameworks.

For each original app in a "baseline" directory, there is a slightly modified app in a corresponding "instrumented" directory which is a copy of the original app with minimal changes to add Sentry instrumentation (error and performance monitoring). The "baseline" apps are implementations from [TechEmpower Framework Benchmarks](https://github.com/TechEmpower/FrameworkBenchmarks).

The benchmark runner (`sentry-sdk-benchmark` tool) takes one or more apps as input and creates, serially, Docker Compose projects to test and gather data to compare baseline and instrumented apps. The output is presented as an HTML report.

Each Docker Compose project is responsible for spinning up a target app, a database server, and auxiliary tools to generate load and measure latencies and resource consumption. Those components exist as a handful of Docker containers:

```
   ┌────────────────────────────────────────────────────────────────────────┐
   │                                                                        │
   │ Container Metrics Collector                                            │
   │         (cAdvisor)                                                     │
   │                                                                        │
   └────────────────────────────────────────────────────────────────────────┘
   ┌────────┐ ┌──────────────┐ ┌───────────────────────┐ ┌──────────────────┐
   │        │ │              │ │                       │ │                  │
   │ Target │ │   Database   │ │    Load Generator,    │ │   Mock Sentry    │
   │  App   │ │ (PostgreSQL) │ │    Data Collector     │ │ Ingestion Server │
   │        │ │              │ │ and Test Orchestrator │ │   (fakerelay)*   │
   │        │ │              │ │      (loadgen)        │ │                  │
   │        │ │              │ │                       │ │                  │
   └────────┘ └──────────────┘ └───────────────────────┘ └──────────────────┘
                                                          * only for
                                                            instrumented apps
```

Every app under test is supposed to interact with a PostgreSQL database in response to requests from the load generator as described in the [Database Updates test](https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview#database-updates).

The load generator throws traffic at the target app at a fixed rate simulating an "open model", as described in [Closed versus open system models and their impact on performance and scheduling, Schroeder et al](https://www.cs.cmu.edu/~bianca/nsdi06.pdf).

When the target app is instrumented with Sentry, the Sentry SDK is configured to send data from the app to the Mock Sentry Ingestion Server, which is basically a custom test replacement for the real Sentry ingestion pipeline.

The load generator is also responsible for orchestrating all test steps and collecting data from all other components (either directly or indirectly via the Container Metrics Collector).

## Usage

You will need a recent version of `docker` (with [Docker Compose V2](https://docs.docker.com/compose/cli-command/#installing-compose-v2)) and `go` (v1.17 or later).

1. Compile the benchmark runner:

    ```shell
    go build .
    ```

2. (Optional) Install the `sentry-sdk-benchmark` binary by moving it to a directory in your `$PATH`, for example:

    ```shell
    mv sentry-sdk-benchmark /usr/local/bin/
    ```

3. Run the `sentry-sdk-benchmark` tool:

    ```shell
    sentry-sdk-benchmark platform/python/django
    ```

## Cleaning Up Resources

The `sentry-sdk-benchmark` tool always tries to clean up resources (containers, images and networks) after running. In the eventual case that something was left behind, the following commands can help cleaning up resources.

Use the commands below with care as some of them may affect resources that were not necessarily created by `sentry-sdk-benchmark`.

<details>
<summary>Docker clean up commands</summary>

<blockquote>
List and remove all Docker Compose projects, including containers, images, and networks:

```shell
docker compose ls -a -q | xargs -tI '{}' docker compose -p '{}' down --remove-orphans --rmi local
```

List and remove all Docker containers:

```shell
docker ps -a -q | xargs -tn10 docker rm -f
```

Remove all unused Docker networks:

```shell
docker network prune
```

Remove images with `sentry-sdk-benchmark` label:

```shell
docker images -f "label=io.sentry.sentry-sdk-benchmark" -q | sort -u | xargs -tn10 docker rmi -f
```

Remove all dangling (untagged) images:

```shell
docker images -f "dangling=true" -q | xargs -tn10 docker rmi -f
```

</blockquote>
</details>

## Adding More Platforms

See [platform/README.md](platform/README.md).

## Licensing
This repostitory may contain code from https://github.com/nylas/nylas-perftools, which is licensed under the following license:
> The MIT License (MIT)
>
> Copyright (c) 2014 Nylas
>
> Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
>
> The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.
>
> THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
