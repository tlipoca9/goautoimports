package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tlipoca9/yevna"
	"github.com/tlipoca9/yevna/parser"
	"github.com/tlipoca9/yevna/tracer"
	"github.com/urfave/cli/v2"
)

var (
	pkgRegex           = regexp.MustCompile(`^package\s+(\w+)$`)
	importsRegex       = regexp.MustCompile(`^import\s*\(\s*$`)
	importOneLineRegex = regexp.MustCompile(`^import\s+".+"$`)
	y                  = yevna.New()
	verbose            bool
)

func init() {
	y.Use(
		yevna.ErrorHandler(),
		yevna.Recover(),
	)
	y.Tracer(tracer.Discard)
}

type GoFile struct {
	Imports []string `json:"imports"`
	Path    string   `json:"path"`
}

func GetGoFiles(moduleName string) ([]GoFile, error) {
	var tmpl bytes.Buffer
	tmpl.WriteString("{{ range .GoFiles }}")
	tmpl.WriteString(fmt.Sprintf("{{ if eq $.Name \"%s\" }}", moduleName))
	tmpl.WriteString(`{{ printf "- path: \"%s/%s\"\n" $.Dir . }}`)
	tmpl.WriteString(`{{ printf "  imports:\n" }}`)
	tmpl.WriteString(`{{ range $.Imports }}`)
	tmpl.WriteString(`{{ printf "    - %s\n" . }}`)
	tmpl.WriteString(`{{ end }}`)
	tmpl.WriteString("{{ end }}")
	tmpl.WriteString("{{ end }}")

	var gofiles []GoFile
	err := y.Run(
		context.Background(),
		yevna.Exec("go", "list", "-f", tmpl.String(), "./..."),
		yevna.Unmarshal(parser.YAML(), &gofiles),
	)
	if err != nil {
		return nil, err
	}
	return gofiles, nil
}

// Filter filters the files with the same directory
func Filter(moduleName string, files []GoFile) []GoFile {
	mainFile := fmt.Sprintf("%s.go", moduleName)

	visit := make(map[string]GoFile)
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		name := filepath.Base(file.Path)

		if name == mainFile {
			visit[dir] = file
		} else if _, ok := visit[dir]; !ok {
			visit[dir] = file
		}

	}

	var ret []GoFile
	for _, v := range visit {
		ret = append(ret, v)
	}

	return ret
}

func GetMissingImports(files []GoFile, pkgs []string) map[string][]string {
	imports := make(map[string][]string)
	for _, file := range files {
		for _, pkg := range pkgs {
			if !slices.Contains(file.Imports, pkg) {
				imports[pkg] = append(imports[pkg], file.Path)
			}
		}
	}
	return imports
}

func AddImport(file string, pkg string) error {
	var found bool
	return y.Run(
		context.Background(),
		yevna.Cat(file),
		yevna.Sed(func(_ int, line string) string {
			if found {
				return line
			}
			if importOneLineRegex.MatchString(line) {
				found = true
				p, _ := strings.CutPrefix(line, "import")
				p = strings.TrimSpace(p)
				var buf bytes.Buffer
				buf.WriteString("import (\n")
				buf.WriteString(fmt.Sprintf("\t_ \"%s\"\n", pkg))
				buf.WriteString(fmt.Sprintf("\t%s\n", p))
				buf.WriteString(")")
				return buf.String()
			}
			if importsRegex.MatchString(line) {
				found = true
				return fmt.Sprintf("%s\n\t_ \"%s\"", line, pkg)
			}
			return line
		}),
		yevna.Sed(func(_ int, line string) string {
			if found {
				return line
			}
			if pkgRegex.MatchString(line) {
				found = true
				return fmt.Sprintf("%s\nimport _ \"%s\"", line, pkg)
			}
			return line
		}),
		yevna.WriteFile(file),
	)
}

func AutoImports(moduleName string, pkgs []string, dryrun bool) error {
	gofiles, err := GetGoFiles(moduleName)
	if err != nil {
		return err
	}

	if verbose {
		fmt.Printf("module '%s' has %d go files\n", moduleName, len(gofiles))
		for _, file := range gofiles {
			fmt.Printf(" - path: %s\n", file.Path)
			fmt.Printf("   imports:\n")
			for _, imp := range file.Imports {
				fmt.Printf("    - %s\n", imp)
			}
		}
	}

	gofiles = Filter(moduleName, gofiles)

	if verbose {
		fmt.Printf("module '%s' has %d go files after filtering\n", moduleName, len(gofiles))
		for _, file := range gofiles {
			fmt.Printf(" - path: %s\n", file.Path)
			fmt.Printf("   imports:\n")
			for _, imp := range file.Imports {
				fmt.Printf("    - %s\n", imp)
			}
		}
	}

	missingImports := GetMissingImports(gofiles, pkgs)

	for pkg, files := range missingImports {
		fmt.Printf("package %s is missing in the following files:\n", pkg)
		for _, file := range files {
			fmt.Printf(" - %s\n", file)
		}
		if dryrun {
			continue
		}
		for _, file := range files {
			err = AddImport(file, pkg)
			if err != nil {
				fmt.Printf("failed to add %s to %s: %v\n", pkg, file, err)
			} else {
				fmt.Printf("added %s to %s\n", pkg, file)
			}
		}
	}

	fmt.Println("goautoimports completed, please run 'go mod tidy' to clean up the imports.")

	return nil
}

func main() {
	app := &cli.App{
		Name:  "goautoimports",
		Usage: "automatically add imports to go files",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "verbose",
				DefaultText: "false",
				Value:       false,
			},
			&cli.StringFlag{
				Name:        "module",
				DefaultText: "main",
				Value:       "main",
				Aliases:     []string{"m"},
			},
			&cli.StringFlag{
				Name:        "pkg",
				DefaultText: "go.uber.org/automaxprocs,github.com/KimMachineGun/automemlimit",
				Value:       "go.uber.org/automaxprocs,github.com/KimMachineGun/automemlimit",
				Aliases:     []string{"p"},
			},
			&cli.BoolFlag{
				Name:        "dryrun",
				DefaultText: "false",
				Value:       false,
			},
		},
		Action: func(c *cli.Context) error {
			verbose = c.Bool("verbose")
			pkgStr := c.String("pkg")
			return AutoImports(
				c.String("module"),
				strings.Split(pkgStr, ","),
				c.Bool("dryrun"),
			)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}

}
