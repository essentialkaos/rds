package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil/panel"
	"github.com/essentialkaos/ek/v13/protip"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// showTip shows tip
func showTip() bool {
	protip.Probability = 0.1
	protip.Options = append(protip.Options, panel.INDENT_OUTER)

	fmtc.NameColor("exec", "{#110}")
	fmtc.NameColor("more", "{s-}")
	fmtc.NameColor("cmd", "{y}")
	fmtc.NameColor("rcmd", "{#160}")
	fmtc.NameColor("opt", "{g}")

	protip.Add(
		&protip.Tip{
			Title:   `Official RDS FAQ`,
			Message: `Don't know how to do something? Checkout official RDS FAQ on {_}https://kaos.sh/rds/w/FAQ{!}`,
			Weight:  0.75,
		},
		&protip.Tip{
			Title: `Mark instance with tags`,
			Message: `You can tag your instances with {?cmd}tag-add{!} command:

{?exec}sudo rds tag-add 1 testing{!}

{?more}More information: sudo rds help tag-add{!}`,
		},
		&protip.Tip{
			Title: `Using colors with tags`,
			Message: `By default, {?cmd}tag-add{!} creates a gray tag. But you can use other colors. You can add
red tag:

{?exec}sudo rds tag-add 1 r:prod{!}

Or green:

{?exec}sudo rds tag-add 1 g:prod{!}

{?more}More information: sudo rds help tag-add{!}`,
		},
		&protip.Tip{
			Title: `View a list of your own instances`,
			Message: `You can view list of your instalces with {?cmd}list{!} command:

{?exec}sudo rds list my{!}

You can also combine it with others filters. For example, you can list your active
instances with {s}#testing{!} tag:

{?exec}sudo rds list my works @testing{!}

{?more}More information: sudo rds help list{!}`,
		},
		&protip.Tip{
			Title: `Redirect command output with pipe`,
			Message: `Some commands like {?cmd}list{!} or {?cmd}info{!} print data in a raw format when command output is
redirected to another application via pipes. For example, you can see a simple list
of your instances ID's redirecting output to {?exec}more{!} or {?exec}less{!}:

{?exec}sudo rds list | more{!}

Or create scripts with it:

{?exec}for id in $(sudo rds list my) ; do sudo rds -y stop $id ; done{!}`,
		},
		&protip.Tip{
			Title: `Track instance metrics`,
			Message: `When you deploy your service, you may want to view basic instance metrics in
interactive mode. You can use {?cmd}track{!} command to do this:

{?exec}sudo rds track 42{!}

{?more}More information: sudo rds help track{!}`,
		},
		&protip.Tip{
			Title: `Don't stop instance before destroy`,
			Message: `There is no need to stop the instance if you want to destroy it. If instance is
running, {?cmd}destroy{!} command will immediately kill it before destroying it.

{?more}More information: sudo rds help destroy{!}`,
		},
		&protip.Tip{
			Title: `Instance for cache without saving data`,
			Message: `If you don't mind about saving instance data, you can create instance with disabled
saves using {?opt}--disable-saves{!} option with {?cmd}create{!} command:

{?exec}sudo rds create --disable-saves{!}

{?more}More information: sudo rds help create{!}`,
		},
		&protip.Tip{
			Title: `Create instance with tags`,
			Message: `You can create and mark instance with tags using {?opt}--tags{!} option with {?cmd}create{!} command:

{?exec}sudo rds create --tags r:important,b:staging{!}

{?more}More information: sudo rds help create{!}`,
		},
		&protip.Tip{
			Title: `Create instance backup`,
			Message: `Before some destructive actions, it's a good idea to make a backup in case something
goes wrong. You can create a backup of your instance data using {?cmd}backup-create{!} command:

{?exec}sudo rds backup-create 42{!}

{?more}More information: sudo rds help backup-create{!}`,
		},
		&protip.Tip{
			Title: `Copy-paste friendly output`,
			Message: `If you want to share some output, give a try to {?opt}--simple{!}/{?opt}-S{!} option. With this option
RDS will simplify the command output. For example:

{?exec}sudo rds -S info 42 instance{!}`,
			Weight: 0.6,
		},
		&protip.Tip{
			Title: `Find the cause of problems`,
			Message: `If your instance isn't working as expected, there are a few commands that can help
you find the cause:

• {?cmd}stats-latency{!}
• {?cmd}stats-error{!}
• {?cmd}slowlog-get{!}

Use {?cmd}help{!} command to find more information about how to use them.`,
		},
		&protip.Tip{
			Title: `Password-protected instances`,
			Message: `You can create a password-protected instance {s}(requires authentication with Redis{!}
{?rcmd}AUTH{s} command){!} using {?opt}--secure{!}/{?opt}-s{!} option with {?cmd}create{!} command:

{?exec}sudo rds settings create --secure{!}

{?more}More information: sudo rds help create{!}`,
		},
		&protip.Tip{
			Title: `View instance clients list`,
			Message: `You can view the list of clients connected to your instance with {?cmd}clients{!} command:

{?exec}sudo rds clients 42{!}

You can also filter clients by name or IP:

{?exec}sudo rds clients 42 192.168.1.123{!}

{?more}More information: sudo rds help clients{!}`,
		},
		&protip.Tip{
			Title: `Interactive and non-interactive Redis CLI`,
			Message: `By default, {?cmd}cli{!} command runs the CLI in interactive mode. But you can also specify
Redis command as part of {?cmd}cli{!} command arguments. In this case, {?cmd}cli{!} executes the command
and returns the result to the terminal.

{?exec}sudo rds cli 42 SET test ABCD{!}
{?exec}sudo rds cli 42 GET test{!}

{?more}More information: sudo rds help cli{!}`,
		},
		&protip.Tip{
			Title: `CLI with admin privileges`,
			Message: `Default Redis user does not have permissions for {*}@admin{!} and {*}@dangerous{!} commands. For
CLI with admin privileges, use {?cmd}cli{!} command with {?opt}--private{!}/{?opt}-p{!} option:

{?exec}sudo rds cli -p 42:2 SAVE ASYNC{!}

{?more}More information: sudo rds help cli{!}`,
		},
		&protip.Tip{
			Title: `View metrics that have changed over time`,
			Message: `You can dump metrics to the file and compare them to the current metrics. Save current
metrics with {?cmd}top-dump{!} command:

{?exec}sudo rds top-dump ~/rds-top-%Y-%m-%d-%H%M.gz{!}

After some time, compare the metrics using {?cmd}top-diff{!} command:

{?exec}sudo rds top-diff ~/rds-top-*.gz connected_clients{!}

{?more}More information: sudo rds help top-dump{!}`,
		},
		&protip.Tip{
			Title: `Pass password from file into command`,
			Message: `You can pass your password using standard input. Save your password to the file. Next,
pass it into the command:

{?exec}cat ~/password.txt | sudo rds cli -p 42{!}

or

{?exec}sudo rds cli -p 42 < ~/password.txt{!}

{y}▲ Don't use echo to pass your password, because other users can see it with{!}
{y}  applications like ps/top/htop and it will be saved into history file.{!}`,
		},
		&protip.Tip{
			Title: `Compare file and in-memory configurations`,
			Message: `User can change instance configuration with {?rcmd}CONFIG SET{!} Redis command. You can view
changed properties using {?cmd}conf{!} command:

{?exec}sudo rds conf 42{!}

{?more}More information: sudo rds help conf{!}`,
		},
		&protip.Tip{
			Title: `Export stats to different format`,
			Message: `You can change output format of {?cmd}stats{!} command using {?opt}--format{!} option:

{?exec}sudo rds stats --format text > ~/stats.txt{!}
{?exec}sudo rds stats --format json > ~/stats.json{!}
{?exec}sudo rds stats --format xml > ~/stats.xml{!}

{?more}More information: sudo rds help stats{!}`,
			Weight: 0.2,
		},
		&protip.Tip{
			Title: `Validate templates after changes`,
			Message: `There is no need to regenerate configuration file or create test instance to check
Redis or Sentinel configuration templates. Just use {?cmd}validate-templates{!} command:

{?exec}sudo rds validate-templates{!}

{?more}More information: sudo rds help validate-templates{!}`,
			Weight: 0.2,
		},
		&protip.Tip{
			Title: `Create many instances at once (superuser only)`,
			Message: `You can create many instances at once using {?cmd}batch-create{!} command with a simple
CSV file.

{?more}More information: sudo rds help batch-create{!}`,
			Weight: 0.2,
		},
		&protip.Tip{
			Title: `Maintenance mode`,
			Message: `When performing some dangerous actions such as Redis update or RDS reconfiguration,
you can restrict user actions using {?cmd}maintenance{!} command:

{?exec}sudo rds maintenance yes{!}

{?more}More information: sudo rds help maintenance{!}`,
		},
		&protip.Tip{
			Title: `Saving and restoring instances state (superuser only)`,
			Message: `You can save and restore instances state {s}(stopped/working){!} using {?cmd}state-save{!} and
{?cmd}state-restore{!} commands:

{?exec}sudo rds state-save ~/states.dat{!}

Stop or start some instances…

{?exec}sudo rds state-restore ~/states.dat{!}

{?more}More information: sudo rds help state-save{!}`,
		},
		&protip.Tip{
			Title: `View and validate RDS settings`,
			Message: `You can view the current RDS settings by using {?cmd}settings{!} command:

{?exec}sudo rds settings{!}

Or view specific settings section:

{?exec}sudo rds settings replcation{!}

{?more}More information: sudo rds help settings{!}`,
		},
		&protip.Tip{
			Title: `Print extra info about instances with list command`,
			Message: `You can view extra information about instances by using {?cmd}list{!} command
with {?opt}-x{!}/{?opt}--extra{!} option:

{?exec}sudo rds list --extra{!}

{?more}More information: sudo rds help list{!}`,
		},
		&protip.Tip{
			Title: `View logs`,
			Message: `You can view RDS CLI, RDS Sync Daemon, or instance log by using {?cmd}log{!} command:

{?exec}sudo rds log 42{!}

{?more}More information: sudo rds help log{!}`,
		},
		&protip.Tip{
			Title: `User-specific preferences`,
			Message: `RDS users can set their own preferences using a configuration file stored in their
home directory. It can be used to disable ProTips, enable Powerline font support,
enable a simplified user interface, or enable automatic paging for long output.

{?more}More information: {_}https://kaos.sh/rds/w/User-preferences{!}`,
		},
	)

	return protip.Show(false)
}
