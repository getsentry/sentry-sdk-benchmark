# Sentry SDK Benchmark

This repository contains reproducible benchmarks to evaluate the instrumentation overhead of Sentry SDKs.

The focus is on Performance Monitoring (tracing) of web servers.

## Usage

You will need `docker` and `go`.

```
go install .
```

```
sentry-sdk-benchmark platform/python/django
```

## How It Works

The `sentry-sdk-benchmark` command automates the following steps:

1. Start `baseline` app:

    ```
    (cd platform/python/django/baseline && docker compose -p django-baseline up -d --build)
    ```

2. Run load generator as a warm-up step:

    ```
    <<<'GET http://localhost:8080/update?query=10' vegeta attack -duration 10s -rate 500/1s | vegeta report
    ```

    You may repeat the command above a couple of times, observing that after the first run the maximum latency reported should drop down.

    Check that the success ratio is 100%.

3. Run load generator to collect benchmark results:

    ```
    <<<'GET http://localhost:8080/update?query=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee results/django100
    ```

4. Tear down containers:

    ```
    (cd platform/python/django/baseline && docker compose -p django-baseline down)
    ```

5. Repeat the steps above, now for the `instrumented` app:

    ```
    (cd platform/python/django/instrumented && docker compose -p django-instrumented up -d --build)
    ```

    ```
    <<<'GET http://localhost:8080/update?query=10' vegeta attack -duration 10s -rate 500/1s | vegeta report
    ```

    ```
    <<<'GET http://localhost:8080/update?query=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee results/django100-instrumented
    ```

    ```
    (cd platform/python/django/instrumented && docker compose -p django-instrumented down)
    ```

6. Open [`plotFiles.html`](tool/plot-hdr-histogram/plotFiles.html) and plot a latency by percentile distribution graph using the two resulting files.
