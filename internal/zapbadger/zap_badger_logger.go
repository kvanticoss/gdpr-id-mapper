package zapbadger

import (
	"go.uber.org/zap"
)

// LoggerBridge adds missing methods to zap.SuggerLogger to match badger.Logger interface
type LoggerBridge struct {
	*zap.SugaredLogger
}

// Warningf decorates zap with Warningf to match badger.Logger interface
func (l *LoggerBridge) Warningf(template string, args ...interface{}) {
	l.Warnf(template, args...)
}
