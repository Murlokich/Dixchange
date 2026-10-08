package pkg

import "github.com/sirupsen/logrus"

type Logger interface {
	Infof(string, ...any)
	Warnf(string, ...any)
	Error(args ...interface{})
	Errorf(string, ...any)
	WithField(string, interface{}) Logger
	WithFields(map[string]any) Logger
}

type LogrusAdapter struct {
	*logrus.Entry
}

func NewLogrusAdapter(logger *logrus.Logger) *LogrusAdapter {
	return &LogrusAdapter{logrus.NewEntry(logger)}
}

func (l *LogrusAdapter) WithField(key string, value interface{}) Logger {
	return &LogrusAdapter{l.Entry.WithField(key, value)}
}

func (l *LogrusAdapter) WithFields(fields map[string]any) Logger {
	return &LogrusAdapter{l.Entry.WithFields(fields)}
}
