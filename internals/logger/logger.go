package logger

import "github.com/sirupsen/logrus"

type Logger interface {
	Info(message string, args ...interface{})
	Error(message string, args ...interface{})
	Debug(message string, args ...interface{})
	Warn(message string, args ...interface{})
}

type LogrusLogger struct {
	logger *logrus.Logger
}

func NewLogrusLogger() *LogrusLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	return &LogrusLogger{logger: logger}
}

func (l *LogrusLogger) Info(message string, args ...interface{}) {
	l.logger.WithFields(logrus.Fields{"args": args}).Info(message)
}

func (l *LogrusLogger) Error(message string, args ...interface{}) {
	l.logger.WithFields(logrus.Fields{"args": args}).Error(message)
}

func (l *LogrusLogger) Debug(message string, args ...interface{}) {
	l.logger.WithFields(logrus.Fields{"args": args}).Debug(message)
}

func (l *LogrusLogger) Warn(message string, args ...interface{}) {
	l.logger.WithFields(logrus.Fields{"args": args}).Warn(message)
}
