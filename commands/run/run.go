package run

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gogf/gf-cli/commands/swagger"
	"github.com/gogf/gf-cli/library/mlog"
	"github.com/gogf/gf/container/garray"
	"github.com/gogf/gf/container/gtype"
	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/net/ghttp"
	"github.com/gogf/gf/os/gcmd"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/os/gfsnotify"
	"github.com/gogf/gf/os/gproc"
	"github.com/gogf/gf/os/gtime"
	"github.com/gogf/gf/os/gtimer"
	"github.com/gogf/gf/text/gstr"
)

type App struct {
	File    string // Go run file name/path.
	Options string // Extra "go run" options.
	Args    string // Auto parse and pack swagger files.
	Swagger bool   // Auto parse and pack swagger files.
}

const (
	gPROXY_CHECK_TIMEOUT = time.Second
	nodeNameInConfigFile = "gfcli.run" // nodeNameInConfigFile is the node name for compiler configurations in configuration file.
)

var (
	process    *gproc.Process
	httpClient = ghttp.NewClient()
)

func init() {
	httpClient.SetTimeout(gPROXY_CHECK_TIMEOUT)
}

func Help() {
	mlog.Print(gstr.TrimLeft(`
USAGE
    gf run FILE [OPTION]

ARGUMENT
    FILE    building file path.
    OPTION  the same options as "go run"/"go build" except some options as follows defined

OPTION
    -/--args     custom process arguments.
    -/--swagger  auto parse and pack swagger into packed/data-swagger.go before running. 

EXAMPLES
    gf run main.go
    gf run main.go --swagger
    gf run main.go --args "server -p 8080"
    gf run main.go -mod=vendor

DESCRIPTION
    The "run" command is used for running go codes with hot-compiled-like feature,
    which compiles and runs the go codes asynchronously when codes change.
`))
}

func Run() {
	parser, err := gcmd.Parse(g.MapStrBool{
		"args": true,
	})
	if err != nil {
		mlog.Fatal(err)
	}
	mlog.SetHeaderPrint(true)
	file := gcmd.GetArg(2)
	if len(file) < 1 {
		mlog.Fatal("file path cannot be empty")
	}
	app := &App{
		File: file,
	}
	// ================================================================================
	// This command is very special that it supports options of "go run" and "go build"
	// from the third parameter of os.Args. That means, we should filter any parameter
	// that "go run" and "go build" do not allow.
	// ================================================================================
	// Swagger checks.
	array := garray.NewStrArrayFrom(os.Args)
	index := array.Search("--swagger")
	if index < 0 {
		index = array.Search("-swagger")
	}
	if index != -1 {
		app.Swagger = true
		array.Remove(index)
	}
	// args checks.
	args := parser.GetOpt("args")
	if args != "" {
		app.Args = args
		index := -1
		array.Iterator(func(k int, v string) bool {
			if gstr.Contains(v, "-args") {
				index = k
				return false
			}
			return true
		})
		if index != -1 {
			v, _ := array.Get(index)
			if gstr.Contains(v, "=") {
				array.Remove(index)
			} else {
				array.Remove(index)
				array.Remove(index)
			}
		}
	}
	// -y checks
	array.RemoveValue("-y")
	array.RemoveValue("--y")
	if array.Len() > 3 {
		app.Options = strings.Join(array.SubSlice(3), " ")
	}
	dirty := gtype.NewBool()
	reloadIgnores := make([]string, 0)
	watchDirs := make([]string, 0)
	recursive := false
	if g.Cfg().Available() {
		reloadIgnores = g.Cfg().GetStrings(fmt.Sprintf("%s.%s", nodeNameInConfigFile, "reloadignore"))
	}
	if len(reloadIgnores) > 0 {
		err = filepath.Walk(gfile.RealPath("."), func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() {
				return nil
			}
			// skip hidden directories like .git, .idea, etc.
			if strings.HasPrefix(filepath.Base(path), ".") {
				return filepath.SkipDir
			}
			for _, pattern := range reloadIgnores {
				if isIgnoreReload(pattern, path) {
					return filepath.SkipDir
				}
			}
			watchDirs = append(watchDirs, path)
			return nil
		})
	} else {
		watchDirs = append(watchDirs, gfile.RealPath("."))
		recursive = true
	}
	if err != nil {
		mlog.Fatal(err)
	}
	for _, dir := range watchDirs {
		_, err = gfsnotify.Add(dir, func(event *gfsnotify.Event) {
			if gfile.ExtName(event.Path) != "go" {
				return
			}
			// Ignore swagger file.
			if gfile.Basename(event.Path) == "data-swagger.go" {
				return
			}
			// Variable <dirty> is used for running the changes only one in one second.
			if !dirty.Cas(false, true) {
				return
			}
			// Ignore file from configurations
			for _, pattern := range reloadIgnores {
				if isIgnoreReload(pattern, event.Path) {
					return
				}
			}
			// With some delay in case of multiple code changes in very short interval.
			gtimer.SetTimeout(1500*gtime.MS, func() {
				defer dirty.Set(false)
				mlog.Printf(`go file changes: %s`, event.String())
				app.Run()
			})
		}, recursive)
		if err != nil {
			mlog.Fatal(err)
		}
	}
	go app.Run()
	select {}
}

func (app *App) Run() {
	// Rebuild and run the codes.
	renamePath := ""
	mlog.Printf("build: %s", app.File)
	outputPath := gfile.Join("bin", gfile.Name(app.File))
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
		if gfile.Exists(outputPath) {
			renamePath = outputPath + "~"
			if err := gfile.Rename(outputPath, renamePath); err != nil {
				mlog.Print(err)
			}
		}
	}
	// Auto swagger.
	if app.Swagger {
		if err := gproc.ShellRun(`gf swagger`); err != nil {
			return
		}
		if gfile.Exists("swagger") {
			packCmd := fmt.Sprintf(`gf pack %s packed/%s -n packed -y`, "swagger", swagger.PackedGoFileName)
			mlog.Print(packCmd)
			if err := gproc.ShellRun(packCmd); err != nil {
				return
			}
		}
	}
	// In case of `pipe: too many open files` error.
	// Build the app.
	buildCommand := fmt.Sprintf(`go build -o %s %s %s`, outputPath, app.Options, app.File)
	mlog.Print(buildCommand)
	result, err := gproc.ShellExec(buildCommand)
	if err != nil {
		mlog.Printf("build error: \n%s%s", result, err.Error())
		return
	}
	// Kill the old process if build successfully.
	if process != nil {
		if err := process.Kill(); err != nil {
			mlog.Debugf("kill process error: %s", err.Error())
			//return
		}
	}
	// Run the binary file.
	runCommand := fmt.Sprintf(`%s %s`, outputPath, app.Args)
	mlog.Print(runCommand)
	if runtime.GOOS == "windows" {
		// Special handling for windows platform.
		// DO NOT USE "cmd /c" command.
		process = gproc.NewProcess(runCommand, nil)
	} else {
		process = gproc.NewProcessCmd(runCommand, nil)
	}
	if pid, err := process.Start(); err != nil {
		mlog.Printf("build running error: %s", err.Error())
	} else {
		mlog.Printf("build running pid: %d", pid)
	}
}

func isIgnoreReload(pattern, path string) bool {
	if pattern == "" {
		return false
	}

	negatePattern := false

	// Handle [Rule 4] which negates the match for patterns leading with "!"
	if pattern[0] == '!' {
		negatePattern = true
		pattern = pattern[1:]
	}

	// Handle [pattern 2, 4], when # or ! is escaped with a \
	// Handle [pattern 4] once we tag negatePattern, strip the leading ! char
	if regexp.MustCompile(`^(\#|\!)`).MatchString(pattern) {
		pattern = pattern[1:]
	}

	// If we encounter a foo/*.blah in a folder, prepend the / char
	if regexp.MustCompile(`([^\/+])/.*\*\.`).MatchString(pattern) && pattern[0] != '/' {
		pattern = "/" + pattern
	}

	// Handle escaping the "." char
	pattern = regexp.MustCompile(`\.`).ReplaceAllString(pattern, `\.`)

	magicStar := "#$~"

	// Handle "/**/" usage
	if strings.HasPrefix(pattern, "/**/") {
		pattern = pattern[1:]
	}
	pattern = regexp.MustCompile(`/\*\*/`).ReplaceAllString(pattern, `(/|/.+/)`)
	pattern = regexp.MustCompile(`\*\*/`).ReplaceAllString(pattern, `(|.`+magicStar+`/)`)
	pattern = regexp.MustCompile(`/\*\*`).ReplaceAllString(pattern, `(|/.`+magicStar+`)`)

	// Handle escaping the "*" char
	pattern = regexp.MustCompile(`\\\*`).ReplaceAllString(pattern, `\`+magicStar)
	pattern = regexp.MustCompile(`\*`).ReplaceAllString(pattern, `([^/]*)`)

	// Handle escaping the "?" char
	pattern = strings.Replace(pattern, "?", `\?`, -1)

	pattern = strings.Replace(pattern, magicStar, "*", -1)

	// Temporary regex
	var expr = ""
	if strings.HasSuffix(pattern, "/") {
		expr = pattern + "(|.*)$"
	} else {
		expr = pattern + "(|/.*)$"
	}
	if strings.HasPrefix(expr, "/") {
		expr = "^(|/)" + expr[1:]
	} else {
		expr = "^(|.*/)" + expr
	}

	reg, err := regexp.Compile(expr)
	if err != nil {
		mlog.Print(err)
		return false
	}

	matched := reg.MatchString(path)
	if matched && !negatePattern {
		return true
	}
	return false
}
