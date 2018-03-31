package smtp

import "github.com/rs/zerolog"

type logHook struct{}

// Run implements a zerolog hook that updates the SMTP warning/error expvars.
func (h logHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	switch level {
	case zerolog.WarnLevel:
		expWarnsTotal.Add(1)
	case zerolog.ErrorLevel:
		expErrorsTotal.Add(1)
	}
}
