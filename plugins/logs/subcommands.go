package logs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dokku/dokku/plugins/common"
)

// CommandDefault displays recent log output
func CommandDefault(appName string, num int64, process string, tail, quiet bool) error {
	if err := common.VerifyAppName(appName); err != nil {
		return err
	}

	if !common.IsDeployed(appName) {
		return fmt.Errorf("App %s has not been deployed", appName)
	}

	s := common.GetAppScheduler(appName)
	t := strconv.FormatBool(tail)
	q := strconv.FormatBool(quiet)
	n := strconv.FormatInt(num, 10)

	_, err := common.CallPlugnTrigger(common.PlugnTriggerInput{
		Args:               []string{s, appName, process, t, q, n},
		DisableStdioBuffer: true,
		StreamStdio:        true,
		Trigger:            "scheduler-logs",
	})
	return err
}

// CommandFailed shows the last failed deploy logs
func CommandFailed(appName string, allApps bool) error {
	if allApps {
		return common.RunCommandAgainstAllAppsSerially(GetFailedLogs, "logs:failed")
	}

	if err := common.VerifyAppName(appName); err != nil {
		return err
	}

	return GetFailedLogs(appName)
}

// CommandReport displays a logs report for one or more apps
func CommandReport(appName string, format string, infoFlag string) error {
	if len(appName) == 0 {
		apps, err := common.DokkuApps()
		if err != nil {
			if errors.Is(err, common.NoAppsExist) {
				common.LogWarn(err.Error())
				return nil
			}
			return err
		}
		for _, appName := range apps {
			if err := ReportSingleApp(appName, format, infoFlag); err != nil {
				return err
			}
		}
		return nil
	}

	return ReportSingleApp(appName, format, infoFlag)
}

// CommandSet sets or clears a logs property for an app
func CommandSet(appName string, property string, value string) error {
	if err := validateSetValue(appName, property, value); err != nil {
		return err
	}

	common.CommandPropertySet("logs", appName, property, value, DefaultProperties, GlobalProperties)

	vectorProperties := map[string]bool{
		"app-label-alias": true,
		"vector-sink":     true,
	}

	if _, ok := vectorProperties[property]; ok {
		common.LogVerboseQuiet(fmt.Sprintf("Writing updated vector config to %s", filepath.Join(common.GetDataDirectory("logs"), "vector.json")))
		return writeVectorConfig()
	}
	return nil
}

// CommandVectorLogs tails the log output for the vector container
func CommandVectorLogs(lines int, tail bool) error {
	if !common.ContainerExists(vectorContainerName) {
		return errors.New("Vector container does not exist")
	}

	if !common.ContainerIsRunning(vectorContainerName) {
		common.LogWarn("Vector container is not running")
	}

	common.LogInfo1Quiet("Vector container logs")
	common.LogVerboseQuietContainerLogsTail(vectorContainerName, lines, tail)

	return nil
}

// CommandVectorStart starts a new vector container
// or starts an existing one if it already exists
func CommandVectorStart(vectorImage string) error {
	common.LogInfo2("Starting vector container")
	common.LogVerbose("Ensuring vector configuration exists")
	if err := writeVectorConfig(); err != nil {
		return err
	}

	if vectorImage == "" {
		vectorImage = common.PropertyGetDefault("logs", "--global", "vector-image", getComputedVectorImage())
	}

	common.LogVerbose("Starting vector container")
	if err := startVectorContainer(vectorImage); err != nil {
		return err
	}

	common.LogVerbose("Waiting for 10 seconds")
	if err := common.ContainerWaitTilReady(vectorContainerName, 10*time.Second); err != nil {
		return errors.New("Vector container did not start properly, run logs:vector-logs for more details")
	}

	common.LogVerbose("Vector container is running")
	return nil
}

// CommandVectorStop stops and removes an existing vector container
func CommandVectorStop() error {
	common.LogInfo2Quiet("Stopping and removing vector container")
	return stopVectorContainer()
}
