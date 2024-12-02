package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"os/exec"
	"text/template"
	"time"

	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/panel"
	"github.com/essentialkaos/ek/v13/spinner"
	"github.com/essentialkaos/ek/v13/system/container"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/input"

	CORE "github.com/essentialkaos/rds/core"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	_GUIDE_TUNING = `{*}1. Tune the kernel memory{!}

Under high-performance conditions, we noticed the occasional blip in performance
due to memory allocation. It turns out this was a known issue with transparent
hugepages.

To fix this issue we have created a profile and script for {?app}tuned{!}.

You can check them here:

{s}•{!} {?cfg}/etc/tuned/no-thp/tuned.conf{!}
{s}•{!} {?cfg}/etc/tuned/no-thp/no-defrag.sh{!}

To enable this profile use the next command:

{?cmd}sudo tuned-adm profile no-thp{!}

{*}2. Tune the kernel{!}

To ensure that Redis handles a large number of connections in a high-performance 
environment tuning the following kernel parameters is recommended.

We have created a configuration with recommended parameters. You can check it
here:

{s}•{!} {?cfg}/etc/sysctl.d/50-rds.conf{!}

To apply these parameters use the next command:

{?cmd}sudo sysctl -p /etc/sysctl.d/50-rds.conf{!}

{*}3. Set file descriptor limits{!}

We have created a configuration with recommended limits. You can check it here:

{s}•{!} {?cfg}/etc/security/limits.d/50-rds.conf{!}

These limits will be applied automatically, so there are no additional actions
to do.

{*}4. Add sudoers configuration{!}

RDS requires sudo privileges for all commands, so you will need to configure
{?app}sudoers{!} for it users.

We have created a configuration for {?prop}rds{!} user group. You can check it here:

{s}•{!} {?cfg}/etc/sudoers.d/rds{!}

Note that this configuration will only work if {?cfg}/etc/sudoers{!} includes files from
{?cfg}/etc/sudoers.d{!}. It must contain this line:

{s}#includedir /etc/sudoers.d{!}

{s-}More info about system configuration can be found at {_}https://redis.io/topics/admin{!}`

	_GUIDE_SYNCING = `{*}1. Generate authentication token{!}

Authentication token used for minions authentication on the master node. You can generate
a new token using the next command:

{?cmd}sudo rds {{.CommandGenToken}}{!}

{*}2. Add master configuration{!}

Open file {?cfg}{{.Config}}{!} on the master node with your favorite editor
and go to the {?prop}[replication]{!} section. Update the next properties:

{s}•{!} {?prop}role{!} to {?prop}master{!};
{s}•{!} {?prop}master-ip{!} to a machine IP address;
{s}•{!} {?prop}auth-token{!} insert previously generated token.

{*}3. Add minion configuration{!}

Open file {?cfg}{{.Config}}{!} on the minion node with your favorite editor
and go to the {?prop}[replication]{!} section. Update the next properties:

{s}•{!} {?prop}role{!} to {?prop}minion{!};
{s}•{!} {?prop}master-ip{!} to a machine IP address;
{s}•{!} {?prop}auth-token{!} insert previously generated token.

{*}4. Start RDS Sync daemon on the master node{!}

Start daemon using the next command:

{?cmd}sudo systemctl start {{.SyncDaemonName}}{!}

Open the daemon log file {?cfg}{{.LogDir}}/rds-sync.log{!} and check
it for errors or warnings. If it has some, fix it and try to start the daemon again.

{y}▲ Note that RDS Sync and all Redis instances MUST not be accessible over the internet.{!}

{*}5. Start RDS Sync daemon on the minion node{!}

Go to minion node and start daemon using the next command:

{?cmd}sudo systemctl start {{.SyncDaemonName}}{!}

Open the daemon log file {?cfg}{{.LogDir}}/rds-sync.log{!} and check
it for errors or warnings. If it has some, fix it and try to start the daemon again.

{*}6. Check the replication status{!}

Check the replication status on master or minion node using the next CLI command:

{?cmd}sudo rds {{.CommandReplication}}{!}

It will show information about master and all connected minion nodes.`
)

const (
	_GUIDE_TUNING_URL  = "https://kaos.sh/rds/w/System-tuning-guide"
	_GUIDE_SYNCING_URL = "https://kaos.sh/rds/w/Syncing-configuration-guide"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// GoCommand is "go" command handler
func GoCommand(args CommandArgs) int {
	generateSUAuthData()

	fmtc.AddColor("app", "{c*}")
	fmtc.AddColor("cmd", "{#110}")
	fmtc.AddColor("cfg", "{#103}")
	fmtc.AddColor("prop", "{#173}")

	configurationStatus, _ := CORE.GetSystemConfigurationStatus(false)

	if configurationStatus.HasProblems {
		fmtc.Println("\n{*}Before RDS usage, we highly recommend to configure your system.{!}\n")

		ok, err := input.ReadAnswer("Show tuning guide?", "Y")

		if err != nil {
			return EC_ERROR
		}

		if ok {
			panel.Panel(
				"GUIDE", "{#63}", "System configuration and tuning for RDS",
				_GUIDE_TUNING, panel.INDENT_OUTER,
			)

			if container.GetEngine() == "" {
				fmtc.Println("Also, we can run all these commands for you as well.\n")
				ok, err = input.ReadAnswer("Run commands?", "Y")

				if ok && err == nil {
					runTuningCommands()
				}
			} else {
				terminal.Warn("You are running RDS inside a container, so we can't configure your system")
				terminal.Warn("for you at this level. You should do it by yourself on the host machine.")
				time.Sleep(5 * time.Second)
			}
		} else {
			fmtc.Printfn(
				"{s}Anyway, you can find system configuration guide at\n{_}%s{!}",
				_GUIDE_TUNING_URL,
			)
		}
	}

	if CORE.IsSyncDaemonInstalled() && CORE.Config.Is(CORE.REPLICATION_AUTH_TOKEN, "") {
		fmtc.Println("\n{*}You have installed and not configured RDS Sync daemon on the system.{!}\n")

		ok, err := input.ReadAnswer("Show syncing configuration guide?", "Y")

		if err != nil {
			return EC_ERROR
		}

		guideData := bytes.Buffer{}
		guideTmpl, _ := template.New("").Parse(_GUIDE_SYNCING)
		guideTmpl.Execute(&guideData, map[string]string{
			"CommandGenToken":    COMMAND_GEN_TOKEN,
			"CommandReplication": COMMAND_REPLICATION,
			"Config":             CONFIG_FILE,
			"LogDir":             CORE.Config.GetS(CORE.PATH_LOG_DIR),
			"SyncDaemonName":     "rds-sync",
		})

		if ok && err == nil {
			panel.Panel(
				"GUIDE", "{#63}", "System configuration and tuning for RDS",
				guideData.String(), panel.INDENT_OUTER,
			)

			fmtc.Printfn("{s}More information can be found at {_}%s{!}", _GUIDE_SYNCING_URL)
		} else {
			fmtc.Printfn(
				"{s}Anyway, you can find syncing configuration guide at\n{_}%s{!}",
				_GUIDE_SYNCING_URL,
			)
		}
	}

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// generateSUAuthData generates superuser auth data
func generateSUAuthData() {
	if CORE.HasSUAuth() {
		terminal.Warn("Superuser credentials already generated")
		return
	}

	password, auth, err := CORE.NewSUAuth()

	if err != nil {
		terminal.Error(err)
		return
	}

	err = CORE.SaveSUAuth(auth, false)

	if err != nil {
		terminal.Error(err)
		return
	}

	fmtc.Println("We have generated a password for you. With this password, you can execute {*}any{!} command on {*}any{!} instance.")

	fmtutil.Separator(false)
	fmtc.Printfn("  {*}Your superuser password is:{!} {c@*} %s {!}", password)
	fmtutil.Separator(false)

	terminal.Warn("Warning! We don't save any passwords. Please save this password in a safe place.")

}

// runTuningCommands runs tuning commands
func runTuningCommands() {
	fmtc.NewLine()

	spinner.Show("Applying tuned configuration")

	cmd := exec.Command("tuned-adm", "profile", "no-thp")
	spinner.Done(cmd.Run() == nil)

	spinner.Show("Applying kernel configuration")

	cmd = exec.Command("sysctl", "-p", "/etc/sysctl.d/50-rds.conf")
	spinner.Done(cmd.Run() == nil)
}
