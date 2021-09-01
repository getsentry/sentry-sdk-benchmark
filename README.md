# Sentry SDK Benchmark

This repository contains reproducible benchmarks to evaluate the instrumentation overhead of Sentry SDKs.

The focus is on Performance Monitoring (tracing) of web servers.

## Usage

You will need `docker` and `go`.

```shell
go install .
```

```shell
sentry-sdk-benchmark platform/python/django
```

## How It Works

The `sentry-sdk-benchmark` command automates the following steps:

1. Start `baseline` app:

    ```shell
    (cd platform/python/django/baseline && docker compose -p django-baseline up -d --build)
    ```

2. Run load generator as a warm-up step:

    ```shell
    <<<'GET http://localhost:8080/update?queries=10' vegeta attack -duration 10s -rate 500/1s | vegeta report
    ```

    You may repeat the command above a couple of times, observing that after the first run the maximum latency reported should drop down.

    Check that the success ratio is 100%.

3. Run load generator to collect benchmark results:

    ```shell
    <<<'GET http://localhost:8080/update?queries=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee results/django100
    ```

4. Tear down containers:

    ```shell
    (cd platform/python/django/baseline && docker compose -p django-baseline down)
    ```

5. Repeat the steps above, now for the `instrumented` app:

    ```shell
    (cd platform/python/django/instrumented && docker compose -p django-instrumented up -d --build)
    ```

    ```shell
    <<<'GET http://localhost:8080/update?queries=10' vegeta attack -duration 10s -rate 500/1s | vegeta report
    ```

    ```shell
    <<<'GET http://localhost:8080/update?queries=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee results/django100-instrumented
    ```

    ```shell
    (cd platform/python/django/instrumented && docker compose -p django-instrumented down)
    ```

6. Open [`plotFiles.html`](tool/plot-hdr-histogram/plotFiles.html) and plot a latency by percentile distribution graph using the two resulting files.

## Manually Cleaning Up Resources

The `sentry-sdk-benchmark` tool always tries to clean up resources (containers and networks) after running. There are failure modes that may leave containers or networks behind. The following two commands can help cleaning up resources. Use with care as they will affect all Docker containers/networks, even those not created by `sentry-sdk-benchmark`.

List and remove all Docker Compose projects, including images:

```shell
for name in $(docker compose ls -q); do docker compose --project-name $name down --remove-orphans --rmi local; done
```

List and remove all Docker containers:

```shell
docker rm -f $(docker ps -a -q)
```

Remove all unused Docker networks:

```shell
docker network prune
```

Remove images with `sentry-sdk-benchmark` label:

```shell
docker rmi $(docker images -f "label=io.sentry.sentry-sdk-benchmark" -q)
```

Remove all dangling (untagged) images:

```shell
docker rmi $(docker images -f "dangling=true" -q)
```

## How To Add More Platforms

See [platform/README.md](platform/README.md).
