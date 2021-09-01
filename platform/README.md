# Benchmark Platforms

This directory contains web applications copied from the [TechEmpower Framework Benchmarks (TFB)](https://github.com/TechEmpower/FrameworkBenchmarks), under their [license](../LICENSE.TechEmpower).

The directory structure is:

- `LANGUAGE/FRAMEWORK/baseline`: copied from TFB with no code changes. Unnecessary files removed.
- `LANGUAGE/FRAMEWORK/instrumented`: copy of `baseline`, with code changes to add Sentry instrumentation (error and performance monitoring).

## Add New Platforms

1. Clone the [TechEmpower Framework Benchmarks](https://github.com/TechEmpower/FrameworkBenchmarks) repository:

    ```zsh
    git clone --depth=1 git@github.com:TechEmpower/FrameworkBenchmarks.git
    ```

    or

    ```zsh
    git clone --depth=1 https://github.com/TechEmpower/FrameworkBenchmarks.git
    ```

2. Copy files from TFB to `platform/.../baseline/`:

    Set `LANGUAGE` and `FRAMEWORK` below to your wish, depending on what is available in TFB.

    ```zsh
    LANGUAGE=Go
    FRAMEWORK=go-std
    mkdir platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/
    rsync -avz FrameworkBenchmarks/frameworks/${LANGUAGE}/${FRAMEWORK}/ platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/
    ```

    Note that we use lower case directory names, while TFB uses a mix of cases. `${LANGUAGE:l}` is a way to tell `zsh` to output a variable in lower case.

    Feel free to use whatever tools you are comfortable with.

3. Delete all `*.dockerfile` files except for one that uses the PostgresQL database. The `sentry-sdk-benchmark` tool expects to find and use a single Dockerfile in the app directory, and will automatically provide a PostgresQL database.

4. Create a `config.json` file.

    ```zsh
    cp platform/python/django/config.json platform/${LANGUAGE:l}/${FRAMEWORK:l}/config.json
    ```

    Open the new file and adjust the configuration as necessary, checking that the target port and path match the app implementation.

    You can refer to the `benchmark_config.json` file copied from the TFB repository or the source code of the app.

    We are interested in the path that implements the [Database Updates test](https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview#database-updates).

5. Delete unnecessary files that are not required for the app to run. For example, most apps include `README.md`, `benchmark_config.json` and `config.toml` that can be safely deleted.

    ```zsh
    rm platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline/{README.md,benchmark_config.json,config.toml}
    ```

    Use a command like `tree` to review the files you are left with.

6. Test the app.

    Use `sentry-sdk-benchmark compose` to generate a Docker compose file that can be used to manually start the app.

    ```zsh
    sentry-sdk-benchmark compose platform/${LANGUAGE:l}/${FRAMEWORK:l}/baseline | docker compose -f - up
    ```

    If everything goes well, you should see the load generator container (`loadgen-baseline-xxxxx`) exit with code `0`. If something goes wrong, look at container logs to debug. Check that the values defined in the `config.json` file match the implementation of the app.

    Press `Ctrl-C` to stop the containers of the Docker compose project, and then run `docker compose down` to remove the containers and related resources.

    Make adjustments as needed and repeat until success.
