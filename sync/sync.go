package sync

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/req"
	"github.com/essentialkaos/ek/v12/signal"
	"github.com/essentialkaos/ek/v12/system/procname"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/man"

	"github.com/essentialkaos/rds/support"

	CORE "github.com/essentialkaos/rds/core"
	MASTER "github.com/essentialkaos/rds/sync/master"
	MINION "github.com/essentialkaos/rds/sync/minion"
	SENTINEL "github.com/essentialkaos/rds/sync/sentinel"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "RDS Sync"
	VER  = "1.1.2"
	DESC = "Syncing daemon for RDS"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Path to conf file
const CONFIG_FILE = "/etc/rds.knf"

// Log file name
const LOG_FILE = "rds-sync.log"

// Command line options
const (
	OPT_NO_COLOR        = "nc:no-color"
	OPT_HELP            = "h:help"
	OPT_VERSION         = "v:version"
	OPT_VERBOSE_VERSION = "vv:verbose-version"

	OPT_GENERATE_MAN = "generate-man"
)

// Exit codes
const (
	EC_OK    = 0
	EC_ERROR = 1
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap is map with options data
var optMap = options.Map{
	OPT_NO_COLOR:        {Type: options.BOOL},
	OPT_HELP:            {Type: options.BOOL},
	OPT_VERSION:         {Type: options.MIXED},
	OPT_VERBOSE_VERSION: {Type: options.BOOL},

	OPT_GENERATE_MAN: {Type: options.BOOL},
}

// colors of app and version
var colorTagApp, colorTagVer string

// ////////////////////////////////////////////////////////////////////////////////// //

// Init is main function
func Init(gitRev string, gomod []byte) {
	runtime.GOMAXPROCS(4)

	preConfigureUI()
	parseOptions()

	if options.Has(OPT_GENERATE_MAN) {
		genManPage()
		os.Exit(EC_OK)
	}

	configureUI()

	if options.GetB(OPT_VERSION) {
		genAbout(gitRev).Print(options.GetS(OPT_VERSION))
		os.Exit(EC_OK)
	}

	if options.GetB(OPT_HELP) {
		genUsage().Print()
		os.Exit(EC_OK)
	}

	initRDSCore()

	if options.GetB(OPT_VERBOSE_VERSION) {
		support.Print(APP, VER, gitRev, gomod)
		os.Exit(EC_OK)
	}

	setupLogger()
	validateConfig()
	addSignalHandlers()
	disableProxy()

	req.Global.SetUserAgent("RSS", VER)
	log.Aux(strings.Repeat("-", 88))

	checkSystemConfiguration()
	renameProcess()

	ec := startSyncDaemon(gitRev)

	log.Aux("Shutdown…")

	shutdown(ec)
}

// preConfigureUI configure user interface
func preConfigureUI() {
	term := os.Getenv("TERM")

	fmtc.DisableColors = true

	if term != "" {
		switch {
		case strings.Contains(term, "xterm"),
			strings.Contains(term, "color"),
			term == "screen":
			fmtc.DisableColors = false
		}
	}

	// Check for output redirect using pipes
	if fsutil.IsCharacterDevice("/dev/stdin") &&
		!fsutil.IsCharacterDevice("/dev/stdout") &&
		os.Getenv("FAKETTY") == "" {
		fmtc.DisableColors = true
	}

	if os.Getenv("NO_COLOR") != "" {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{#DC382C}", "{#A32422}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{#160}", "{#124}"
	default:
		colorTagApp, colorTagVer = "{r}", "{r}"
	}
}

// configureUI configure user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}
}

// parseOptions parses command-line options
func parseOptions() {
	_, errs := options.Parse(optMap)

	if len(errs) == 0 {
		return
	}

	printError("Options parsing errors: ")

	for _, err := range errs {
		printError("  %v\n", err)
	}

	os.Exit(EC_ERROR)
}

// initRDSCore initializes RDS core
func initRDSCore() {
	errs := CORE.Init(CONFIG_FILE)

	if len(errs) == 0 {
		return
	}

	for _, err := range errs {
		// At this moment all error messages goes to stderr
		printError(err.Error())
	}

	os.Exit(EC_ERROR)
}

// setupLogger setup logger
func setupLogger() {
	err := CORE.SetLogOutput(LOG_FILE, CORE.Config.GetS(CORE.LOG_LEVEL), true)

	if err != nil {
		printError("Can't setup logger: %v", err)
		os.Exit(EC_ERROR)
	}
}

// addSignalHandlers add signal handlers for TERM/INT/HUP signals
func addSignalHandlers() {
	signal.Handlers{
		signal.TERM: termSignalHandler,
		signal.INT:  intSignalHandler,
		signal.HUP:  hupSignalHandler,
	}.Track()
}

// disableProxy disable proxy for requests to sync daemon
func disableProxy() {
	os.Setenv("http_proxy", "")
	os.Setenv("https_proxy", "")
	os.Setenv("HTTP_PROXY", "")
	os.Setenv("HTTPS_PROXY", "")
}

// checkSystemConfiguration check system configuration
func checkSystemConfiguration() {
	status, err := CORE.GetSystemConfigurationStatus(false)

	if err == nil && !status.HasProblems {
		return
	}

	if err != nil {
		log.Crit("Can't check system configuration: %v", err)
		CORE.Shutdown(EC_ERROR)
	}

	if status.HasTHPIssues {
		log.Crit("You should disable THP (Transparent Huge Pages) on this system.")
	}

	if status.HasLimitsIssues {
		log.Crit("You should increase the maximum number of open file descriptors for the Redis user.")
	}

	if status.HasKernelIssues {
		log.Crit("You should set vm.overcommit_memory setting and increase net.core.somaxconn setting in your sysctl configuration file.")
	}

	if status.HasFSIssues {
		log.Crit("You should increase the size of the partition used for Redis data. Size of the partition should be at least twice bigger than the size of available memory (physical + swap).")
	}

	CORE.Shutdown(EC_ERROR)
}

// validateConfig validate auth token
func validateConfig() {
	if !CORE.Config.HasProp(CORE.REPLICATION_AUTH_TOKEN) {
		log.Crit("Auth token not defined in %s", CORE.REPLICATION_AUTH_TOKEN)
		CORE.Shutdown(EC_ERROR)
	}

	if len(CORE.Config.GetS(CORE.REPLICATION_AUTH_TOKEN)) != CORE.TOKEN_LENGTH {
		log.Crit("Auth token have wrong size")
		CORE.Shutdown(EC_ERROR)
	}
}

// startSyncDaemon starts sync daemon service
func startSyncDaemon(gitRev string) int {
	var ec int // Exit code

	role := CORE.Config.GetS(CORE.REPLICATION_ROLE)

	if role == "" {
		log.Warn("Replication is disabled. Shutdown…")
		return EC_ERROR
	}

	switch role {
	case CORE.ROLE_MASTER:
		ec = MASTER.Start(APP, VER, gitRev)
	case CORE.ROLE_MINION:
		ec = MINION.Start(APP, VER, gitRev)
	case CORE.ROLE_SENTINEL:
		ec = SENTINEL.Start(APP, VER, gitRev)
	default:
		log.Crit("Unknown sync daemon role %s", role)
		return EC_ERROR
	}

	return ec
}

// renameProcess renames current daemon process
func renameProcess() {
	var args []string
	var role string

	switch CORE.Config.GetS(CORE.REPLICATION_ROLE) {
	case CORE.ROLE_MASTER, CORE.ROLE_MINION, CORE.ROLE_SENTINEL:
		role = CORE.Config.GetS(CORE.REPLICATION_ROLE)
	default:
		return
	}

	for i := range os.Args {
		switch i {
		case 0:
			args = append(args, fmt.Sprintf("rds-sync:%s", role))
		default:
			args = append(args, "")
		}
	}

	procname.Set(args)
}

// shutdown gracefully shutdown daemon
func shutdown(code int) {
	switch CORE.Config.GetS(CORE.REPLICATION_ROLE) {
	case CORE.ROLE_MASTER:
		MASTER.Stop()
	case CORE.ROLE_MINION:
		MINION.Stop()
	case CORE.ROLE_SENTINEL:
		SENTINEL.Stop()
	}

	log.Info("Bye-Bye!")

	CORE.Shutdown(code)
}

// printError prints error message to console
func printError(f string, a ...any) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// termSignalHandler is handler for TERM signal
func termSignalHandler() {
	log.Info("Got TERM signal, shutdown…")
	shutdown(EC_OK)
}

// intSignalHandler is handler for INT signal
func intSignalHandler() {
	log.Info("Got INT signal, shutdown…")
	shutdown(EC_OK)
}

// hupSignalHandler is handler for HUP signal
func hupSignalHandler() {
	log.Info("Got HUP signal, log will be reopened…")
	CORE.ReopenLog()
	log.Info("Log was reopened by HUP signal")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// genManPage generates man page for app
func genManPage() {
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo()

	info.AppNameColorTag = "{*}" + colorTagApp

	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VERSION, "Show information about version")

	return info
}

// genAbout generates basic info about app
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER + "/" + CORE.VERSION,
		Desc:    DESC,
		Year:    2009,

		AppNameColorTag: "{*}" + colorTagApp,
		VersionColorTag: colorTagVer,

		License:    "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
		Owner:      "ESSENTIAL KAOS",
		BugTracker: "https://kaos.sh/rds",
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	return about
}
