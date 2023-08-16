package command

import (
	"bytes"
	"fmt"
	"github.com/gogf/gf/v2/text/gregex"
	cli "github.com/jawher/mow.cli"
	"github.com/samber/lo"
	"github.com/yashen/go-pkg-graph/internal"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

import "github.com/gogf/gf/v2/os/gfile"

type _Dep struct {
	Name     string
	DepNames []string
}

var optOutputFile string
var optDepth int
var optIncludePrefix []string
var optExcludePrefix []string

func Main() {
	if len(os.Args) == 2 && os.Args[1] == "install" {
		selfPath := gfile.SelfPath()
		targetPath := path.Join(os.Getenv("GOPATH"), "bin", "go-pkg-graph")
		if runtime.GOOS == "windows" {
			targetPath = targetPath + ".exe"
		}
		gfile.Copy(selfPath, targetPath)
		fmt.Println(fmt.Sprintf("current has installed to %s", targetPath))
		return
	}

	cliApp := cli.App("go-pkg-graph", "gen package depend graph")
	cliApp.StringOptPtr(&optOutputFile, "o output", "graph.svg", "output svg file")
	cliApp.StringsOptPtr(&optIncludePrefix, "i include", nil, "package include prefix")
	cliApp.StringsOptPtr(&optExcludePrefix, "e exclude", nil, "package exclude prefix")
	cliApp.IntOptPtr(&optDepth, "d depth", 3, "package depth")
	modulePathOpt := cliApp.StringOpt("m module", ".", "module path")
	cliApp.Action = func() {

		modulePath, err := filepath.Abs(*modulePathOpt)
		if err != nil {
			panic(err)
		}
		moduleFile := path.Join(modulePath, "go.mod")
		moduleName := getModuleName(moduleFile)

		deptList := make([]_Dep, 0)

		filepath.Walk(modulePath, func(path string, info fs.FileInfo, err error) error {
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			packageName, deps := parseFileDep(path)

			parts := strings.Split(path[len(modulePath)+1:], "\\")
			parts = parts[:len(parts)-1]
			if len(parts) == 0 {
				parts = []string{moduleName}
			}
			parts = append([]string{moduleName}, parts...)
			if parts[len(parts)-1] != packageName {
				parts[len(parts)-1] = packageName
			}

			packageName = strings.Join(parts, "/")

			deptList = append(deptList, _Dep{
				Name:     packageName,
				DepNames: deps,
			})

			return nil
		})

		genImage(moduleName, deptList)

	}
	cliApp.Run(os.Args)
}

func genImage(moduleName string, depList []_Dep) {

	arrayList := make([][2]string, 0)

	for _, dep := range depList {

		if strings.HasSuffix(dep.Name, "_test") {
			continue
		}

		if strings.Contains(dep.Name, "/.") {
			continue
		}

		for _, name := range dep.DepNames {
			if !strings.HasPrefix(name, moduleName) {
				continue
			}

			arrayList = append(arrayList, [2]string{dep.Name, name})

		}
	}

	for i, _ := range arrayList {
		item := arrayList[i]
		leftParts := strings.Split(item[0], "/")
		if len(leftParts) > optDepth {
			leftParts = leftParts[:optDepth]
		}
		rightParts := strings.Split(item[1], "/")
		if len(rightParts) > optDepth {
			rightParts = rightParts[:optDepth]
		}
		item = [2]string{strings.Join(leftParts, "/"), strings.Join(rightParts, "/")}
		arrayList[i] = item

	}

	if len(optExcludePrefix) > 0 {
		arrayList = lo.Filter(arrayList, func(item [2]string, index int) bool {
			return !internal.Any(item[0:], func(item string) bool {
				return internal.Any(optExcludePrefix, func(prefix string) bool {
					return strings.HasPrefix(item, prefix)
				})
			})
		})
	}

	if len(optIncludePrefix) > 0 {
		arrayList = lo.Filter(arrayList, func(item [2]string, index int) bool {
			return internal.Both(item[0:], func(item string) bool {
				return internal.Any(optIncludePrefix, func(prefix string) bool {
					return strings.HasPrefix(item, prefix)
				})
			})
		})
	}

	arrayList = lo.Filter(arrayList, func(item [2]string, index int) bool {
		return item[0] != item[1]
	})
	arrayList = lo.Uniq(arrayList)

	var buider strings.Builder
	buider.WriteString("digraph G {\n")
	for _, item := range arrayList {
		buider.WriteString(fmt.Sprintf("\"%s\"->\"%s\"", item[0], item[1]))

		if strings.HasPrefix(item[0], item[1]+"/") {
			buider.WriteString(" [color=red]")
			fmt.Println(fmt.Sprintf("%s depend on %s", item[0], item[1]))
		}

		buider.WriteString(";\n")
	}
	buider.WriteString("\n}")

	content := buider.String()

	if err := runDotToImageCallSystemGraphviz(optOutputFile, content); err != nil {
		panic(err)
	}

}

func getModuleName(moduleFile string) string {

	var result string
	prefix := "module "
	gfile.ReadLines(moduleFile, func(line string) error {
		if result != "" {
			return nil
		}
		if strings.HasPrefix(line, prefix) {
			result = strings.TrimSpace(line[len(prefix):])
		}
		return nil
	})
	return result
}

const packagePrefix = "package "
const importPrefix = "import "
const importRangeStart = "import ("
const importRangeEnd = ")"

func parseFileDep(filePath string) (string, []string) {
	importRange := false
	var packageName string
	deps := make([]string, 0)
	gfile.ReadLines(filePath, func(line string) error {
		if packageName == "" {
			if strings.HasPrefix(line, packagePrefix) {
				packageName = line[len(packagePrefix):]
			}
			return nil
		}

		if importRange {
			if strings.HasPrefix(line, importRangeEnd) {
				importRange = false
			} else {
				if line == "" {
					return nil
				}
				if result := parseImport(line); result != "" {
					deps = append(deps, result)
				}
			}
		} else if strings.HasPrefix(line, importRangeStart) {
			importRange = true
		} else if strings.HasPrefix(line, importPrefix) {
			if result := parseImport(line[len(importPrefix):]); result != "" {
				deps = append(deps, result)
			}
		}
		return nil
	})

	return packageName, deps
}

func parseImport(input string) string {

	result, err := gregex.MatchString(`"([^"]*)"`, input)
	if err != nil {
		panic(err)
	}
	if len(result) < 2 {
		return ""
	}
	return result[1]

}

func runDotToImageCallSystemGraphviz(outfname string, content string) error {
	if !strings.HasSuffix(outfname, ".svg") {
		outfname = outfname + ".svg"
	}
	cmd := exec.Command("dot", "-Tsvg", "-o", outfname)
	cmd.Stdin = bytes.NewReader([]byte(content))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command '%v': %v\n%v", cmd, err, stderr.String())
	}
	return nil
}
