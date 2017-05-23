# Gonfigen

A configuration management tool for your Go project. Takes care of generating the configuration files and loaders.

### Disclaimer: Although this works for the most part, I wouldn't suggest using it as it still is WIP.

## How it works

By adding `//go:generate gonfigen -type=Config` to your configuration struct (where `Config` is the name of said struct) and then running `go generate` you will get a `cmd/gonfig/gonfig.go` command which is used to (non)interactively create and manage a config file (`gonfig.toml` by default) by creating it either directly from the struct itself or from a template distributed with your project. You will also get a `gonfig_loaders.go` generated in your main package, containing the `LoadConfig()` method.

This is the most simplest usage as both gonfigen and gonfig have more flags/customization options that will get documented in time.

## TODO:
- show types (--show-types?)
- load from env (json/xml?)
- readme
- tests
- heavy heavy refactor