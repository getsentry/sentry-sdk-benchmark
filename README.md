# Sentry SDK Benchmark

This repository contains reproducible benchmarks to evaluate the instrumentation overhead of Sentry SDKs.

The focus is on Performance Monitoring (tracing) of web servers.

## Usage

You will need `docker` and [`vegeta`](https://github.com/tsenart/vegeta) or another load generator.

Start apps and dependencies:

```
(cd platform/python/django/baseline && docker compose -p django-baseline up -d --build)
(cd platform/javascript/express/baseline && docker compose -p express-baseline up -d --build)
(cd platform/python/django/instrumented && docker compose -p django-instrumented up -d --build)
(cd platform/javascript/express/instrumented && docker compose -p express-instrumented up -d --build)
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
find platform -type f -name docker-compose.yml -print -execdir docker compose down \;
```

---

Future:

```
./bench platform/python/django
```

The above will:

- [x] 1. Start a stack of containers (avoiding clashing with any existing state / containers) for **baseline** benchmark
- [ ] 2. Run load generator to warm up target web app
- [ ] 3. Concurrently:
    a. Run load generator to benchmark latencies
    b. Collect CPU/memory usage every 1s and remember max
- [ ] 4. Store hdr output from load generator and CPU/memory max usage
- [ ] 5. Generate plot comparing baseline vs instrumented latencies
