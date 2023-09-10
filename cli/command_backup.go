package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/path"
	"github.com/essentialkaos/ek/v12/sortutil"
	"github.com/essentialkaos/ek/v12/spinner"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/timeutil"

	CORE "github.com/essentialkaos/rds/core"
	REDIS "github.com/essentialkaos/rds/redis"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	BACKUP_PERMS = 0600 // Default permissions for backups
	MAX_BACKUPS  = 10   // Maximum number of backups
)

// ////////////////////////////////////////////////////////////////////////////////// //

// BackupCreateCommand is "backup-create" command handler
func BackupCreateCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error("Can't check instance state: %v", err)
		return EC_ERROR
	}

	backupName := fmt.Sprintf("backup-%d.rdb", time.Now().Unix())
	rdbFile := CORE.GetInstanceRDBPath(id)
	backupFile := path.Join(CORE.GetInstanceDataDirPath(id), backupName)

	if state.IsWorks() {
		spinner.Show("Saving instance data in background")

		_, err := CORE.ExecCommand(id, &REDIS.Request{
			Command: []string{"BGSAVE"},
		})

		if err != nil {
			spinner.Done(false)
			fmtc.NewLine()
			terminal.Error("Can't exec BGSAVE command: %v", err)
			return EC_ERROR
		}

		waitForDump(id)
	} else {
		if !fsutil.IsExist(rdbFile) {
			terminal.Warn("There is no RDB snapshot of instance data")
			return EC_ERROR
		}

		spinner.Show("Creating snapshot of instance data")
	}

	if getBackupNum(id) == MAX_BACKUPS {
		err := cleanOldBackups(id, 1)

		if err != nil {
			terminal.Error(err.Error())
			return EC_ERROR
		}
	}

	rdbFileSize := fsutil.GetSize(rdbFile)

	if rdbFileSize > 0 {
		spinner.Update(
			"Copying snapshot of RDB file {s}(%s){!}",
			fmtutil.PrettySize(rdbFileSize),
		)
	} else {
		spinner.Update("Copying snapshot of RDB file")
	}

	err = fsutil.CopyFile(rdbFile, backupFile, BACKUP_PERMS)

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error("Can't copy snapshot file: %v", err)
		return EC_ERROR
	}

	logger.Info(id, "Created RDB backup")

	return EC_OK
}

// BackupRestoreCommand is "backup-restore" command handler
func BackupRestoreCommand(args CommandArgs) int {
	var index int

	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	state, err := CORE.GetInstanceState(id, false)

	if err != nil {
		terminal.Error("Can't check instance state: %v", err)
		return EC_ERROR
	}

	if state.IsWorks() {
		terminal.Warn("Instance must be stopped for restoring data from snapshot")
		return EC_WARN
	}

	numBackups := getBackupNum(id)

	if numBackups == 0 {
		terminal.Warn("There are no snapshots of given instance data")
		return EC_WARN
	}

	listBackups(id)

	fmtc.NewLine()

	for {
		selectedIndex, err := terminal.Read(fmt.Sprintf("Enter index of snapshot to restore (1-%d)", numBackups), true)

		if err != nil {
			return EC_OK
		}

		fmtc.NewLine()

		index, err = strconv.Atoi(selectedIndex)

		if err != nil {
			terminal.Error("Invalid snapshot index: %v\n", err)
			continue
		}

		if index > numBackups || index < 1 {
			terminal.Error("There is no backup with index %d\n", index)
			continue
		}

		break
	}

	snapshots := getBackupFiles(id)

	if len(snapshots) == 0 {
		terminal.Error("Can't get list of snapshots")
		return EC_ERROR
	}

	spinner.Show("Restoring instance data from snapshot")

	err = restoreBackupSnapshot(id, snapshots[index-1])

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error("Can't restore snapshot: %v", err)
		logger.Error(id, "Tried to restore snapshot, but got error: %v", err)
		return EC_ERROR
	}

	logger.Info(id, "Restored RDB backup")

	return EC_OK
}

// BackupCleanCommand is "backup-clean" command handler
func BackupCleanCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	numBackups := getBackupNum(id)

	if numBackups == 0 {
		terminal.Warn("There are no snapshots of given instance data")
		return EC_WARN
	}

	listBackups(id)

	fmtc.NewLine()

	ok, _ := terminal.ReadAnswer("Delete all these snapshots?", "N")

	if !ok {
		return EC_OK
	}

	fmtc.NewLine()

	spinner.Show("Removing instance backups {s}(%d){!}", numBackups)

	err = cleanOldBackups(id, MAX_BACKUPS)

	spinner.Done(err == nil)

	if err != nil {
		fmtc.NewLine()
		terminal.Error("Can't clean backups: %v", err)
		logger.Error(id, "Tried to remove RDB backups (%d), but got error: %v", numBackups, err)
		return EC_ERROR
	}

	logger.Info(id, "Removed RDB backups (%d)", numBackups)

	return EC_OK
}

// BackupListCommand is "backup-list" command handler
func BackupListCommand(args CommandArgs) int {
	err := args.Check(false)

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	id, _, err := CORE.ParseIDDBPair(args.Get(0))

	if err != nil {
		terminal.Error(err.Error())
		return EC_ERROR
	}

	if getBackupNum(id) == 0 {
		terminal.Warn("There are no snapshots of given instance data")
		return EC_WARN
	}

	listBackups(id)

	return EC_OK
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getBackupNum returns number of backups for instance with given ID
func getBackupNum(id int) int {
	return len(getBackupFiles(id))
}

// getBackupFiles returns slice with paths to backup files
func getBackupFiles(id int) []string {
	files := fsutil.List(
		CORE.GetInstanceDataDirPath(id), false,
		fsutil.ListingFilter{MatchPatterns: []string{"backup-*.rdb"}},
	)

	sortutil.StringsNatural(files)

	return files
}

// listBackups shows table with information about all backups of given instance
func listBackups(id int) {
	if getBackupNum(id) == 0 {
		return
	}

	dataDir := CORE.GetInstanceDataDirPath(id)
	t := table.NewTable().SetHeaders("#", "SIZE", "DATE")

	for index, file := range getBackupFiles(id) {
		t.Add(
			fmt.Sprintf("{s}%d{!}", index+1),
			fmtutil.PrettySize(fsutil.GetSize(path.Join(dataDir, file))),
			extractBackupDateFromFilename(file),
		)
	}

	t.Render()
}

// cleanOldBackups remove old backups for instance with given ID
func cleanOldBackups(id, numOldBackups int) error {
	numOldBackups = mathutil.Between(numOldBackups, 1, MAX_BACKUPS)

	dataDir := CORE.GetInstanceDataDirPath(id)

	for index, file := range getBackupFiles(id) {
		err := os.Remove(path.Join(dataDir, file))

		if err != nil {
			return fmt.Errorf("Can't remove backup file: %w", err)
		}

		if index+1 == numOldBackups {
			break
		}
	}

	return nil
}

// restoreBackupSnapshot replaces default dump file with snapshot
func restoreBackupSnapshot(id int, snapshot string) error {
	rdbFile := CORE.GetInstanceRDBPath(id)
	dataDir := CORE.GetInstanceDataDirPath(id)
	snapshotFile := path.Join(dataDir, snapshot)

	uid, gid, err := fsutil.GetOwner(dataDir)

	if err != nil {
		return fmt.Errorf("Can't get data directory owner: %v", err)
	}

	err = fsutil.CopyFile(snapshotFile, rdbFile, 0640)

	if err != nil {
		return fmt.Errorf("Can't copy snapshot: %v", err)
	}

	err = os.Chown(rdbFile, uid, gid)

	if err != nil {
		return fmt.Errorf("Can't change RDB file owner: %v", err)
	}

	return nil
}

// waitForDump waits until RDB file updated
func waitForDump(id int) {
	var tempRDB string

	rdbFile := CORE.GetInstanceRDBPath(id)
	modTime, _ := fsutil.GetMTime(rdbFile)

	for range time.NewTicker(time.Second).C {
		curModTime, _ := fsutil.GetMTime(rdbFile)

		if !curModTime.IsZero() && !curModTime.Equal(modTime) {
			return
		}

		if tempRDB == "" {
			tempRDB = getTemporaryRDBPath(id)
		}

		if tempRDB != "" {
			tempRDBSize := fsutil.GetSize(tempRDB)

			if tempRDBSize > 0 {
				spinner.Update(
					"Saving instance data in background {s}(%s){!}",
					fmtutil.PrettySize(tempRDBSize),
				)
			}
		}
	}
}

// extractBackupDateFromFilename extracts backup creation date from filename
func extractBackupDateFromFilename(file string) string {
	ts := strutil.Exclude(file, "backup-")
	ts = strutil.Exclude(ts, ".rdb")
	tsi, _ := strconv.ParseInt(ts, 10, 64)

	if tsi == 0 {
		return "unknown"
	}

	return timeutil.Format(time.Unix(tsi, 0), "%Y/%m/%d %H:%M:%S")
}

// getTemporaryRDB returns path to temporary RDB file
func getTemporaryRDBPath(id int) string {
	files := fsutil.List(
		CORE.GetInstanceDataDirPath(id), false,
		fsutil.ListingFilter{
			MatchPatterns: []string{"temp-*.rdb"},
			MTimeYounger:  time.Now().Unix() - 60,
		},
	)

	if len(files) == 0 {
		return ""
	}

	return path.Join(CORE.GetInstanceDataDirPath(id), files[0])
}
