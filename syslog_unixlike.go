//go:build !windows && !plan9

package packaged

import (
	"fmt"
	"log/syslog"
	"os"
	"runtime/debug"
)

func logToJournalctl(priority, message string) {
	logger, err := syslog.New(syslog.LOG_DAEMON, ServiceName)
	if err != nil {
		fmt.Printf("Failed to connect to syslog: %v\n", err)
		return
	}
	defer logger.Close()

	fmt.Fprintf(os.Stderr, "%s %s\n ----- Stack ----- \n %s", priority, message, debug.Stack())

	switch priority {
	case "CRITICAL":
		logger.Crit(message)
	case "ERROR":
		logger.Err(message)
	default:
		logger.Info(message)
	}
}
