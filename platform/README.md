# Benchmark Platforms

This directory contains web applications copied from the [TechEmpower Framework Benchmarks (TFB)](https://github.com/TechEmpower/FrameworkBenchmarks), under their [license](../LICENSE.TechEmpower).

The directory structure is:

- `LANGUAGE/FRAMEWORK/baseline`: copied from TFB with no code changes. Unnecessary files removed.
- `LANGUAGE/FRAMEWORK/instrumented`: copy of `baseline`, with code changes to add Sentry instrumentation (error and performance monitoring).
