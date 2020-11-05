package logger

import (
	"accdns/common"
	"fmt"
	"github.com/phachon/go-logger"
	"runtime"
	"strings"
)

var Logger = go_logger.NewLogger()

func Init() error {
	switch common.Config.Log.LogLevelForConsole {
	case "debug":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_DEBUG, &go_logger.ConsoleConfig{})
	case "info":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_INFO, &go_logger.ConsoleConfig{})
	case "warning":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_WARNING, &go_logger.ConsoleConfig{})
	case "error":
		_ = Logger.Detach("console")
		_ = Logger.Attach("console", go_logger.LOGGER_LEVEL_ERROR, &go_logger.ConsoleConfig{})
	case "none":
		_ = Logger.Detach("console")
	default:
		Error("Set Log Level for Console", "unknown log level", common.Config.Log.LogLevelForConsole)
	}

	if common.Config.Log.LogFilePath != "" {
		logFileConfig := &go_logger.FileConfig{
			Filename:  common.Config.Log.LogFilePath,
			MaxSize:   common.Config.Log.LogFileMaxSizeKB,
			DateSlice: "d",
		}
		switch common.Config.Log.LogLevelForFile {
		case "debug":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_DEBUG, logFileConfig)
		case "info":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_INFO, logFileConfig)
		case "warning":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_WARNING, logFileConfig)
		case "error":
			_ = Logger.Attach("file", go_logger.LOGGER_LEVEL_ERROR, logFileConfig)
		case "none":
		default:
			Error("Set Log Level for File", "unknown log level", common.Config.Log.LogLevelForFile)
		}
	}
	return nil
}
func Error(process string, objs ...interface{}) {
	msg := "[" + GetCallerName(1) + "] {" + process + "} "
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Error(strings.TrimSpace(msg))
}
func Alert(process string, objs ...interface{}) {
	msg := "[" + GetCallerName(1) + "] {" + process + "} "
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Alert(strings.TrimSpace(msg))
}
func Warning(process string, objs ...interface{}) {
	msg := "[" + GetCallerName(1) + "] {" + process + "} "
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Warning(strings.TrimSpace(msg))
}
func Info(process string, objs ...interface{}) {
	msg := "[" + GetCallerName(1) + "] {" + process + "} "
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Info(strings.TrimSpace(msg))
}
func Debug(process string, objs ...interface{}) {
	msg := "[" + GetCallerName(1) + "] {" + process + "} "
	for _, obj := range objs {
		msg += fmt.Sprint(obj) + " "
	}
	Logger.Debug(strings.TrimSpace(msg))
}

func GetCallerName(skip int) string {
	pc, _, _, _ := runtime.Caller(skip + 1)
	callerFullName := runtime.FuncForPC(pc).Name()
	callerNameFields := strings.Split(callerFullName, "/")
	return callerNameFields[len(callerNameFields)-1]
}
