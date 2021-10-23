# Benchmark Platforms

This directory contains web applications copied from the [TechEmpower Framework Benchmarks (TFB)](https://github.com/TechEmpower/FrameworkBenchmarks), under their [license](../LICENSE.TechEmpower).

The directory structure is:

- `LANGUAGE/FRAMEWORK/baseline`: copied from TFB with no code changes. Unnecessary files removed.
- `LANGUAGE/FRAMEWORK/instrumented`: copy of `baseline`, with code changes to add Sentry instrumentation (error and performance monitoring).

## Add New Platform or Framework

Follow the steps below to add new platforms or frameworks to use with the `sentry-sdk-benchmark` tool.

*The command line examples below all use [Z shell](https://en.wikipedia.org/wiki/Z_shell) syntax. You are not required to use `zsh`, as the same outcome can be achieved with the tools of your preference.*


1. Clone the [TechEmpower Framework Benchmarks](https://github.com/TechEmpower/FrameworkBenchmarks) repository.

    ```zsh
    git clone --depth=1 git@github.com:TechEmpower/FrameworkBenchmarks.git
    ```

    or, alternatively, using HTTPS:

    ```zsh
    git clone --depth=1 https://github.com/TechEmpower/FrameworkBenchmarks.git
    ```

    We suggest a shallow clone (`--depth=1`) because the repository history is heavy and takes a considerable amount of time to download.

2. Copy files from TFB to `platform/.../baseline/`.

    Set `LANGUAGE` and `FRAMEWORK` below to your wish, depending on what is available in TFB.

    ```zsh
    LANGUAGE=Go
    FRAMEWORK=go-std
    mkdir -p platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/
    rsync -avz FrameworkBenchmarks/frameworks/${LANGUAGE}/${FRAMEWORK}/ platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/
    ```

    Note that we use lower case directory names, while TFB uses a mix of cases. The form `${LANGUAGE:l}` is a way to tell `zsh` to output a variable in lower case.


3. Delete `*.dockerfile` files, except for one that uses the PostgresQL database.

    The `sentry-sdk-benchmark` tool expects to find and use a single Dockerfile in the app directory, and will automatically provide a PostgresQL database.

4. Create a `config.json` file.

    ```zsh
    cp platform/python/django/config.json platform/${LANGUAGE:l}/${FRAMEWORK:l}/config.json
    ```

    Open the new file and adjust the configuration as necessary, checking that the values provided match the app implementation. You can refer to the `benchmark_config.json` file copied from the TFB repository or the source code of the app.

    We are interested in the path that implements the [Database Updates test](https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview#database-updates).

5. Delete unnecessary files.

    We typically do not need all the files from TFB to run the app with Postgres. However, it can be trick to remove parts without breaking the whole. We recommend deleting `README.md`, `benchmark_config.json` and `config.toml` that most apps include.

    However, avoid the temptation to change the app to remove unused parts. For most apps, it would be too much work to try to split out just what is used, for very little if any benefit. Try to keep the code as close to upstream as possible because that makes it easier to apply upstream patches later.

    ```zsh
    rm platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/{README.md,benchmark_config.json,config.toml}
    ```

    Use a command like `tree` to review the files you are left with. Remove the most that you can, but avoid making changes to the files you keep, so that it is easy to use a `diff` tool to compare and sanity check the app against the original source.

6. Test the baseline app.

    Use `sentry-sdk-benchmark` to run the app.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline
    ```

    If everything goes well, you should see the load generator container (`loadgen-baseline-xxxxx`) exit with code `0`. If something goes wrong, look at container logs to debug (saved under `result/`, full path printed on-screen). Check that the values defined in the `config.json` file match the implementation of the app.

    Make adjustments as needed and repeat until success.

7. Copy `baseline` app to `platform/.../instrumented/`:

    ```zsh
    rsync -avz platform/${LANGUAGE:l}/${FRAMEWORK:l}/{baseline,instrumented}/
    ```

    At this point you might stage files to commit later:

    ```zsh
    git add platform
    ```

8. Instrument with Sentry.

    Make minimal edits to the source code to add Sentry tracing instrumentation according to [Sentry's official documentation](https://docs.sentry.io).

    **Do not configure a DSN in the source code**. The `sentry-sdk-benchmark` tool automatically provides a test DSN through an environment variable. Most SDKs will read and use the `SENTRY_DSN` environment variable without need for manual configuration.

    The typical changes are:

    - Add a Sentry SDK to the list of dependencies.
    - Change the main entrypoint of the web app to call `sentry.init` or equivalent, making sure to enable tracing.
    - Add Sentry middlewares as necessary.
    - Both error monitoring and tracing should have their sample rates configured to 100% (`1.0`), unless you are interested in benchmarking alternative configurations.
    - You may enable debug logging while you test things out, but we recommend disabling it before committing.
    - Enable database instrumentation for PostgreSQL.

    You can use `git` to double check your minimal edits:

    ```zsh
    git diff -- platform
    ```

9. Test the instrumented app.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}/instrumented
    ```

    As with the baseline app, make adjustments as needed and repeat until success.

    There should be a transaction for every incoming request and spans for all reads and writes from/to the database.

10. Do a full run with both baseline and instrumented apps.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}
    ```

Review your changes one last time, commit and push. Done!

## Add an OpenTelemetry-instrumented App

Comparing a baseline app and a Sentry-instrumented app gives some insight. Adding a second instrumented app can help put results into perspective, in particular to sanity check that Sentry's instrumentation overhead is similar to that of other libraries.

We've chosen to compare with OpenTelemetry because, among other reasons, there are SDKs available for many platforms and because of its vendor neutrality.

The steps to add an OpenTelemetry-instrumented app are similar to steps 7 to 10 above, reproduced here for convenience:

1. Copy `baseline` app to `platform/.../opentelemetry/`:

    ```zsh
    rsync -avz platform/${LANGUAGE:l}/${FRAMEWORK:l}/{baseline,opentelemetry}/
    ```

    At this point you might stage files to commit later:

    ```zsh
    git add platform
    ```

2. Instrument with OpenTelemetry.

    Make minimal edits to the source code to add OpenTelemetry tracing instrumentation.

    The actual steps will depend on the platform and framework, but these guidelines should be followed:

    - Use a `BatchSpanProcessor`
    - Use a `ZipkinExporter` using HTTP and the default configuration (it should automatically pick up configuration from the `OTEL_EXPORTER_ZIPKIN_ENDPOINT` environment variable)
    - Add PostgreSQL database instrumentation (there should be a span for every database interaction: reads and writes)
    - Add framework-specific instrumentation (there should be at least a span for every incoming request)

    You can use `git` to double check your minimal edits:

    ```zsh
    git diff -- platform
    ```

3. Test the OpenTelemetry-instrumented app.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}/opentelemetry
    ```

4. Do a full run with baseline, Sentry-instrumented and OpenTelemetry-instrumented apps.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}
    ```

Review your changes one last time, commit and push. Done!

## Optimizing For Build Cache

Building Docker images takes a considerable amount of time.

The baseline apps we take from TFB not always follow [the best practices for writing Dockerfiles][1], and that is okay. While we try to keep as close to upstream as possible, we also try to [minimize the time wasted downloading packages from the Internet][2].

When adding a new platform, we prefer to install SDKs and dependencies required for Sentry and OpenTelemetry instrumentation in a way that allows Docker to share as many build layers as possible across baseline and instrumented apps, even if that means we have to do some unconventional steps.

For example:

- [Python](https://github.com/getsentry/sentry-sdk-benchmark/blob/6cefaf0b60989a909a647dc748008c7aa42d2016/platform/python/django/instrumented/django-postgresql.dockerfile#L7-L8)
- [JavaScript](https://github.com/getsentry/sentry-sdk-benchmark/blob/6cefaf0b60989a909a647dc748008c7aa42d2016/platform/javascript/express/instrumented/express-postgres.dockerfile#L7-L10)
- [Ruby](https://github.com/getsentry/sentry-sdk-benchmark/blob/6cefaf0b60989a909a647dc748008c7aa42d2016/platform/ruby/rails/instrumented/rails.dockerfile#L16-L18)

Note, however, that [separating the install steps of the dependencies][3] in common with the baseline app is not always practical. So for some platforms we just accept that and move on (for example [Go](https://github.com/getsentry/sentry-sdk-benchmark/blob/6cefaf0b60989a909a647dc748008c7aa42d2016/platform/go/go-std/instrumented/go-pgx.dockerfile#L9-L10) and [Java](https://github.com/getsentry/sentry-sdk-benchmark/blob/6cefaf0b60989a909a647dc748008c7aa42d2016/platform/java/spring/instrumented/spring-jpa.dockerfile#L4-L5)).

[1]: https://docs.docker.com/develop/develop-images/dockerfile_best-practices/
[2]: https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#leverage-build-cache
[3]: https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#add-or-copy
