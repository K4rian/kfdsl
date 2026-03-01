package base

import (
	"time"
)

type ServiceOptions struct {
	RootDirectory    string
	WorkingDirectory string
	AutoRestart      bool
	MaxRestarts      int
	RestartDelay     time.Duration
	ShutdownTimeout  time.Duration
	KillTimeout      time.Duration
}

/*
func DefaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		AutoRestart:     true,
		MaxRestarts:     settings.DefaultMaxRestarts,
		RestartDelay:    settings.DefaultRestartDelay * time.Second,
		ShutdownTimeout: settings.DefaultShutdownTimeout * time.Second,
		KillTimeout:     settings.DefaultKillTimeout * time.Second,
	}
}
*/
