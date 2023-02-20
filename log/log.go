package log

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/spy16/pgbase/errors"
)

var lg = logrus.New()

// Setup configures the global logger instance with level and
// formatter.
func Setup(level, format string) {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.WarnLevel
	}
	lg.SetLevel(lvl)
	lg.SetFormatter(&logrus.TextFormatter{})
	if format == "json" {
		lg.SetFormatter(&logrus.JSONFormatter{})
	}
}

func Debug(ctx context.Context, msg string, args ...Fields) {
	doLog(ctx, logrus.DebugLevel, msg, nil, args)
}

func Info(ctx context.Context, msg string, args ...Fields) {
	doLog(ctx, logrus.InfoLevel, msg, nil, args)
}

func Warn(ctx context.Context, msg string, args ...Fields) {
	doLog(ctx, logrus.WarnLevel, msg, nil, args)
}

func Error(ctx context.Context, msg string, err error, args ...Fields) {
	doLog(ctx, logrus.ErrorLevel, msg, err, args)
}

func Fatal(ctx context.Context, msg string, err error, args ...Fields) {
	doLog(ctx, logrus.FatalLevel, msg, err, args)
	os.Exit(1)
}

func doLog(ctx context.Context, level logrus.Level, msg string, err error, args []Fields) {
	m := mergeFields(fromCtx(ctx), args)
	if err != nil {
		e := errors.E(err)
		m["error"] = err.Error()
		if len(e.Attribs) > 0 {
			m["error.attribs"] = e.Attribs
		}
	}
	lg.WithFields(m).Logln(level, msg)
}

func mergeFields(base Fields, fields []Fields) map[string]any {
	res := map[string]any{}
	for k, v := range base {
		res[k] = v
	}
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			res[k] = v
		}
	}
	return res
}
