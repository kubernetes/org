# k-license

## Prepends project files with given template.

* Can be used for adding licence or copyright information on src files of project.
* Skip directory, if template (as provided) already present
* Supports Golang/Python source files, Dockerfile, Makefiles and bash scripts

## Build

```
go build -o k-license k-license.go 
```

Example
To Apply header from ./boilerplate
folder default

```
$ go run k-license.go add --help
Add Headers to files

Usage:
  k-license add [flags]

Flags:
      --confirm            
  -e, --exclude strings    comma-separated list of directories to exclude (default [external/bazel_tools,.git,node_modules,_output,third_party,vendor,verify/boilerplate/test])
  -h, --help               help for add
      --path string        Defaults to Current directory (default ".")
      --templates string   directory containing license templates (default "../../hack/boilerplate")

```

## Help

```
$ go run k-license.go --help 
Tool for Adding license Headers

Usage:
  k-license [command]

Available Commands:
  add         Add Headers to files
  help        Help about any command

Flags:
  -h, --help   help for k-license

Use "k-license [command] --help" for more information about a command.

```