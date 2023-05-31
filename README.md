# GitHub Actions Executor

The GitHub Actions Executor is a helper command for running [GitHub Actions](https://docs.github.com/en/actions) from
inside of [Dagger](https://dagger.io) containers.


**Warning**: This project is still in early development and is not ready for use.
**Warning**: This project intended to used with the project [gale](https://github.com/aweris/gale) in [Dagger](https://dagger.io) containers

## Features

- Execute GitHub Actions locally
- Test and Debug GitHub Actions locally without pushing to GitHub
- List previous runs of a workflow and access their logs, artifacts, and metadata

## TODO

- [x] Support for `custom` actions using `node`
- [x] Support for `bash` scripts in `run` steps
- [ ] Support for `docker://` actions
- [ ] Support for github expressions, e.g. `${{ github.ref }}`
- [ ] Support for `secrets`
- [ ] Support for triggers and events
- [ ] Support for `composite` actions
- [ ] Support for reusable workflows

## Installation

```bash
go get github.com/aweris/ghx
```

## Usage

```bash
Github Actions Executor is a helper tool for gale to run workflows locally

Usage:
  ghx [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  run         Runs all configured steps
  version     Print version information
  with        Adds new configuration to execute

Flags:
  -h, --help   help for ghx

Use "ghx [command] --help" for more information about a command.
```

Adding job to run:

```bash
ghx with job --workflow .github/workflows/test.yml --job test
```

help for job:

```bash
Sets workflow and job environment variables and configures to all steps in the job.

Usage:
  ghx with job [flags]

Flags:
  -h, --help              help for job
      --job string        Name of the job
      --workflow string   Name of the workflow. If workflow doesn't have name, than it must be relative path to the workflow file
```

Adding step to run:

```bash
ghx with step --uses actions/checkout@v3 --name checkout --with token=<token> --with ref=master
```

help for step:

```bash
Add new step to execute

Usage:
  ghx with step [flags]

Flags:
      --env stringToString    Environment variable names and values (default [])
  -h, --help                  help for step
      --id string             Unique identifier of the step
      --name string           Name of the step
      --override              Override step if already exists
      --run string            Command to run for the step
      --shell string          Shell to use for the step
      --uses string           Action to run for the step
      --with stringToString   Input names and values for the step (default [])
```

Running configured steps:

```bash
ghx run
```