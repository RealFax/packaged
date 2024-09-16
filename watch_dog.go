package packaged

import (
	"context"
	"fmt"
	"log/syslog"
	"os"
	"runtime/debug"
)

var ServiceName string = "packaged-daemon"

func restartRetry(retry int32, srv Service, logger Logger) (err error) {
	for i := 0; i < int(retry); i++ {
		if err = srv.OnStart(); err != nil {
			if logger != nil {
				logger.Error("packaged: failed to start service.",
					"name", srv.Name(),
					"reason", err,
					"retry", i,
				)
			}
		}
	}
	return err
}

func runBlocking(maxRetry int32, policy Restart, srv Service) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v\n%s", r, debug.Stack())
			logToJournalctl("CRITICAL", fmt.Sprintf("Panic in runBlocking: %v\n%s", r, debug.Stack()))
		}
	}()

	switch policy {
	case RestartIgnore:
		return srv.OnStart()
	case RestartRetry:
		return restartRetry(maxRetry, srv, nil)
	default:
		return nil
	}
}

func runAsync(ctx context.Context, maxRetry int32, policy Restart, srv Service, logger Logger) {
	switch policy {
	case RestartIgnore:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logToJournalctl("CRITICAL", fmt.Sprintf("Panic in runAsync (RestartIgnore): %v\n%s", r, debug.Stack()))
				}
			}()
			if err := srv.OnStart(); err != nil {
				logger.Error("packaged: failed to start service.", "name", srv.Name(), "restart_policy", policy, "reason", err)
				logToJournalctl("ERROR", fmt.Sprintf("Failed to start service: %s, policy: %v, error: %v", srv.Name(), policy, err))
			}
		}()
	case RestartRetry:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logToJournalctl("CRITICAL", fmt.Sprintf("Panic in runAsync (RestartRetry): %v\n%s", r, debug.Stack()))
				}
			}()
			restartRetry(maxRetry, srv, logger)
		}()
	case RestartAlways:
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logToJournalctl("CRITICAL", fmt.Sprintf("Panic in runAsync (RestartAlways): %v\n%s", r, debug.Stack()))
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := srv.OnStart(); err != nil {
						logger.Error(
							"packaged: failed to start service.",
							"name", srv.Name(),
							"restart_policy", policy,
							"reason", err,
						)
						logToJournalctl("ERROR", fmt.Sprintf("Failed to start service: %s, policy: %v, error: %v", srv.Name(), policy, err))
					} else {
						return
					}
				}
			}
		}()
	}
}

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
