package l

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

// 自定义时间编码器
type customTimeEncoder struct {
	zapcore.TimeEncoder
}

func (c customTimeEncoder) EncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("20060102 15:04:05.000"))
}

// 自定义级别编码器
type customLevelEncoder struct {
	zapcore.LevelEncoder
}

func (c customLevelEncoder) EncodeLevel(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("D")
	case zapcore.InfoLevel:
		enc.AppendString("I")
	case zapcore.WarnLevel:
		enc.AppendString("W")
	case zapcore.ErrorLevel:
		enc.AppendString("E")
	case zapcore.FatalLevel:
		enc.AppendString("F")
	case zapcore.PanicLevel:
		enc.AppendString("P")
	default:
		enc.AppendString(l.CapitalString())
	}
}

// 自定义调用者编码器
type customCallerEncoder struct {
	zapcore.CallerEncoder
}

func (c customCallerEncoder) EncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// 去掉包名前缀
	callerPath := caller.TrimmedPath()
	if idx := strings.Index(callerPath, "/"); idx != -1 {
		callerPath = callerPath[idx+1:]
	}
	enc.AppendString(callerPath)
}

func Setup(config Config) error {
	// 设置日志编码器
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = customTimeEncoder{}.EncodeTime
	encoderConfig.EncodeLevel = customLevelEncoder{}.EncodeLevel
	encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
	encoderConfig.EncodeCaller = customCallerEncoder{}.EncodeCaller
	encoderConfig.ConsoleSeparator = " " // 设置控制台输出分隔符为单个空格

	// 设置日志等级
	level := zapcore.DebugLevel // 默认日志等级
	if config != nil && config.Level() != "" {
		switch strings.ToLower(config.Level()) {
		case "debug":
			level = zapcore.DebugLevel
		case "info":
			level = zapcore.InfoLevel
		case "warn", "warning":
			level = zapcore.WarnLevel
		case "error":
			level = zapcore.ErrorLevel
		case "fatal":
			level = zapcore.FatalLevel
		case "panic":
			level = zapcore.PanicLevel
		}
	}

	var writer zapcore.WriteSyncer
	if config != nil {
		// 配置日志轮转
		hook := &lumberjack.Logger{
			Filename:   config.Path(),
			MaxSize:    10, // 每个日志文件最大尺寸，单位MB
			MaxBackups: 30, // 保留的旧日志文件最大数量
			MaxAge:     7,  // 保留的旧日志文件最大天数
			Compress:   true,
		}

		if config.Console() {
			// 创建多输出
			writer = zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(os.Stdout),
				zapcore.AddSync(hook),
			)
		} else {
			writer = zapcore.AddSync(hook)
		}
	} else {
		writer = zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
		)
	}

	if config != nil && config.Async() {
		writer = &zapcore.BufferedWriteSyncer{
			WS:   writer,
			Size: 4096, // 4KB 缓冲区
		}
	}

	// 创建核心
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		writer, level, // 使用配置的日志等级
	)

	// 创建logger，移除默认的字段分隔符
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()

	return nil
}

func Unsetup() error {
	return nil
}

func Debug(args ...interface{}) {
	sugar.Debug(args...)
}

func Info(args ...interface{}) {
	sugar.Info(args...)
}

func Warn(args ...interface{}) {
	sugar.Warn(args...)
}

func Error(args ...interface{}) {
	sugar.Error(args...)
}

func Fatal(args ...interface{}) {
	sugar.Fatal(args...)
}

func Panic(args ...interface{}) {
	sugar.Panic(args...)
}

func Debugf(message string, args ...interface{}) {
	sugar.Debugf(message, args...)
}

func Infof(message string, args ...interface{}) {
	sugar.Infof(message, args...)
}

func Warnf(message string, args ...interface{}) {
	sugar.Warnf(message, args...)
}

func Errorf(message string, args ...interface{}) {
	sugar.Errorf(message, args...)
}

func Fatalf(message string, args ...interface{}) {
	sugar.Fatalf(message, args...)
}

func Panicf(message string, args ...interface{}) {
	sugar.Panicf(message, args...)
}
