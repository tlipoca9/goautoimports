# goautoimports

`goautoimports` is a tool to automatically add missing imports to Go source code.

By default, `goautoimports` will add following imports to `main` package:
- `"go.uber.org/automaxprocs"`
- `"github.com/KimMachineGun/automemlimit"`

## Installation

```bash
go install github.com/tlipoca9/goautoimports@latest
```

## Usage

```bash
goautoimports
```

## Options

```bash
$ goautoimports -h

NAME:
   goautoimports - automatically add imports to go files

USAGE:
   goautoimports [global options] command [command options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose                 (default: false)
   --module value, -m value  (default: main)
   --pkg value, -p value     (default: go.uber.org/automaxprocs,github.com/KimMachineGun/automemlimit)
   --dryrun                  (default: false)
   --help, -h                show help
   
```