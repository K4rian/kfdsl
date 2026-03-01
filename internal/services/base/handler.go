package base

// ServiceLogHandler is a function that receives each log line from the process.
// Return true to signal that the service should be restarted.
type ServiceLogHandler func(line string) (shouldRestart bool)
