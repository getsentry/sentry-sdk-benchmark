# Sentry SDK Benchmark

This repository contains reproducible benchmarks to evaluate the instrumentation overhead of Sentry SDKs.

The focus is on Performance Monitoring (tracing) of web servers.

## Usage

You will need `docker` and [`vegeta`](https://github.com/tsenart/vegeta) or another load generator.

Start app and dependencies:

```
(cd platform/python/django/baseline && docker compose up -d --build)
(cd platform/javascript/express/baseline && docker compose up -d --build)
```

or

```
(cd platform/python/django/instrumented && docker compose up -d --build)
(cd platform/javascript/express/instrumented && docker compose up -d --build)
```

Run load generator:

**Django**

```
# warmup step
<<<'GET http://localhost:8080/update?query=10' vegeta attack -duration 10s -rate 500/1s | vegeta report

<<<'GET http://localhost:8080/update?query=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee django100
```

**Express**

```
<<<'GET http://localhost:8080/updates?query=10' vegeta attack -duration 10s -rate 500/1s | vegeta report

<<<'GET http://localhost:8080/updates?query=100' vegeta attack -duration 20s -rate 500/1s | vegeta report -type=hdrplot | tee express100
```

Plot graphs with [`plotFiles.html`](plotFiles.html).
<!--
https://hdrhistogram.github.io/HdrHistogram/plotFiles.html
-->

Clean up:

```
(cd platform/python/django/baseline && docker compose down)
(cd platform/python/django/instrumented && docker compose down)
```
