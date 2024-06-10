package logging

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ViolationLogger *zap.Logger

func InitViolationLogger(logLevel string) {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateOnly + " " + time.TimeOnly)
	fileEncoder := zapcore.NewJSONEncoder(config)
	os.Mkdir("./logs", os.ModePerm)

	logFile, _ := os.OpenFile("logs/violations.json", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	writer := zapcore.AddSync(logFile)
	defaultLogLevel, _ := zapcore.ParseLevel(zap.WarnLevel.String())
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
	)

	ViolationLogger = zap.New(core, zap.AddStacktrace(zapcore.ErrorLevel))
}
