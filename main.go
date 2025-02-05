package main

import (
	_ "github.com/gogf/gf-cli/boot"

	"fmt"
	"github.com/gogf/gf-cli/command/build"
	"github.com/gogf/gf-cli/command/docker"
	"github.com/gogf/gf-cli/command/env"
	"github.com/gogf/gf-cli/command/fix"
	"github.com/gogf/gf-cli/command/gen"
	"github.com/gogf/gf-cli/command/get"
	"github.com/gogf/gf-cli/command/initialize"
	"github.com/gogf/gf-cli/command/install"
	"github.com/gogf/gf-cli/command/mod"
	"github.com/gogf/gf-cli/command/pack"
	"github.com/gogf/gf-cli/command/run"
	"github.com/gogf/gf-cli/command/swagger"
	"github.com/gogf/gf-cli/command/update"
	"github.com/gogf/gf-cli/library/allyes"
	"github.com/gogf/gf-cli/library/mlog"
	"github.com/gogf/gf-cli/library/proxy"
	"github.com/gogf/gf/errors/gerror"
	"github.com/gogf/gf/os/gbuild"
	"github.com/gogf/gf/os/gcmd"
	"github.com/gogf/gf/os/gfile"
	"github.com/gogf/gf/text/gregex"
	"github.com/gogf/gf/text/gstr"
	"strings"
)

const (
	VERSION = "v1.16.4"
)

func init() {
	// Automatically sets the golang proxy for all commands.
	proxy.AutoSet()
}

var (
	helpContent = gstr.TrimLeft(`
USAGE
    gf COMMAND [ARGUMENT] [OPTION]

COMMAND
    env        show current Golang environment variables
    get        install or update GF to system in default...
    gen        automatically generate go files for ORM models...
    mod        extra features for go modules...
    run        running go codes with hot-compiled-like feature...
    init       create and initialize an empty GF project...
    help       show more information about a specified command
    pack       packing any file/directory to a resource file, or a go file...
    build      cross-building go project for lots of platforms...
    docker     create a docker image for current GF project...
    swagger    swagger feature for current project...
    update     update current gf binary to latest one (might need root/admin permission)
    install    install gf binary to system (might need root/admin permission)
    version    show current binary version info

OPTION
    -y         all yes for all command without prompt ask 
    -?,-h      show this help or detail for specified command
    -v,-i      show version information

ADDITIONAL
    Use 'gf help COMMAND' or 'gf COMMAND -h' for detail about a command, which has '...' 
    in the tail of their comments.
`)
)

func main() {
	defer func() {
		if exception := recover(); exception != nil {
			if err, ok := exception.(error); ok {
				mlog.Print(gerror.Current(err).Error())
			} else {
				panic(exception)
			}
		}
	}()

	allyes.Init()

	command := gcmd.GetArg(1)
	// Help information
	if gcmd.ContainsOpt("h") && command != "" {
		help(command)
		return
	}
	switch command {
	case "help":
		help(gcmd.GetArg(2))
	case "version":
		version()
	case "env":
		env.Run()
	case "get":
		get.Run()
	case "gen":
		gen.Run()
	case "fix":
		fix.Run()
	case "mod":
		mod.Run()
	case "init":
		initialize.Run()
	case "pack":
		pack.Run()
	case "docker":
		docker.Run()
	case "swagger":
		swagger.Run()
	case "update":
		update.Run()
	case "install":
		install.Run()
	case "build":
		build.Run()
	case "run":
		run.Run()
	default:
		for k := range gcmd.GetOptAll() {
			switch k {
			case "?", "h":
				mlog.Print(helpContent)
				return
			case "i", "v":
				version()
				return
			}
		}
		// No argument or option, do installation checks.
		if !install.IsInstalled() {
			mlog.Print("hi, it seams it's the first time you installing gf cli.")
			s := gcmd.Scanf("do you want to install gf binary to your system? [y/n]: ")
			if strings.EqualFold(s, "y") {
				install.Run()
				gcmd.Scan("press <Enter> to exit...")
				return
			}
		}
		mlog.Print(helpContent)
	}
}

// help shows more information for specified command.
func help(command string) {
	switch command {
	case "get":
		get.Help()
	case "gen":
		gen.Help()
	case "init":
		initialize.Help()
	case "docker":
		docker.Help()
	case "swagger":
		swagger.Help()
	case "build":
		build.Help()
	case "pack":
		pack.Help()
	case "run":
		run.Help()
	case "mod":
		mod.Help()
	default:
		mlog.Print(helpContent)
	}
}

// version prints the version information of the cli tool.
func version() {
	info := gbuild.Info()
	if info["git"] == "" {
		info["git"] = "none"
	}
	mlog.Printf(`GoFrame CLI Tool %s, https://goframe.org`, VERSION)
	gfVersion, err := getGFVersionOfCurrentProject()
	if err != nil {
		gfVersion = err.Error()
	} else {
		gfVersion = gfVersion + " in current go.mod"
	}
	mlog.Printf(`GoFrame Version: %s`, gfVersion)
	mlog.Printf(`CLI Installed At: %s`, gfile.SelfPath())
	if info["gf"] == "" {
		mlog.Print(`Current is a custom installed version, no installation information.`)
		return
	}

	mlog.Print(gstr.Trim(fmt.Sprintf(`
CLI Built Detail:
  Go Version:  %s
  GF Version:  %s
  Git Commit:  %s
  Build Time:  %s
`, info["go"], info["gf"], info["git"], info["time"])))
}

// getGFVersionOfCurrentProject checks and returns the GoFrame version current project using.
func getGFVersionOfCurrentProject() (string, error) {
	goModPath := gfile.Join(gfile.Pwd(), "go.mod")
	if gfile.Exists(goModPath) {
		match, err := gregex.MatchString(`github.com/gogf/gf\s+([\w\d\.]+)`, gfile.GetContents(goModPath))
		if err != nil {
			return "", err
		}
		if len(match) > 1 {
			return match[1], nil
		}
		return "", gerror.New("cannot find goframe requirement in go.mod")
	} else {
		return "", gerror.New("cannot find go.mod")
	}
}
