package logger

import (
	"os"
	"strings"

	cfg "github.com/shreyasksrao/jobmanager/app/config"
	"github.com/shreyasksrao/jobmanager/lib/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Global logger instance
var AppLogger, JobManagerLogger, JobRunnerLogger core.Logger
var AppLoggerCleanup, JobManagerLoggerCleanup, JobRunnerLoggerCleanup func()

func Initialize(config *cfg.Config) {
	logLevel := config.LogLevel
	if logLevel != "" {
		logLevel = "INFO"
	}
	AppLogger, AppLoggerCleanup = CreateLogger(config.GetApplicationLogFilePath(), logLevel)
	AppLogger.Infof("Successfully setup the logger for the application.")

	jobManagerLogFilePath := config.GetLogDirectory() + "/" + cfg.JOB_MANAGER_LOG_FILE_NAME
	JobManagerLogger, JobManagerLoggerCleanup = CreateLogger(jobManagerLogFilePath, logLevel)

	jobRunnerLogFilePath := config.GetLogDirectory() + "/" + cfg.JOB_RUNNER_LOG_FILE_NAME
	JobRunnerLogger, JobRunnerLoggerCleanup = CreateLogger(jobRunnerLogFilePath, logLevel)
}

func CleanUpLoggers() {
	AppLoggerCleanup()
	JobManagerLoggerCleanup()
	JobRunnerLoggerCleanup()
}

func CreateLogger(fileName string, level string) (logger core.Logger, cleanup func()) {
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapcore.Level(getLogLevel(level, "ZAP")))

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.CallerKey = "caller"
	encoderConfig.MessageKey = "msg"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder

	fileEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	core := zapcore.NewCore(fileEncoder, zapcore.AddSync(file), atomicLevel)
	zapLogger := zap.New(core)
	cleanup = func() {
		zapLogger.Sync()
	}
	logger = zapLogger.Sugar()
	return
}

func GetAppLogger() core.Logger {
	return AppLogger
}

func GetJobManagerLogger() core.Logger {
	return JobManagerLogger
}

func GetJobRunnerLogger() core.Logger {
	return JobRunnerLogger
}

func getLogLevel(level, loggerLibName string) (logLevel int) {
	if strings.ToUpper(loggerLibName) == "ZAP" {
		switch strings.ToUpper(level) {
		case "DEBUG":
			logLevel = -1
		case "INFO":
			logLevel = 0
		case "WARN":
			logLevel = 1
		case "ERROR":
			logLevel = 2
		}
		return logLevel
	}
	return -1111
}
