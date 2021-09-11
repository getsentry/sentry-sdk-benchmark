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

    You can use `git` to double check your minimal edits:

    ```zsh
    git diff -- platform
    ```

9. Test the instrumented app.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}/instrumented
    ```

    As with the baseline app, make adjustments as needed and repeat until success.

10. Do a full run with both baseline and instrumented apps.

    ```zsh
    sentry-sdk-benchmark platform/${LANGUAGE:l}/${FRAMEWORK:l}
    ```

Review your changes one last time, commit and push. Done!
