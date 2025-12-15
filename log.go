package l

import (
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger   *zap.Logger
	sugar    *zap.SugaredLogger
	initOnce sync.Once
	initMu   sync.RWMutex
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
	// 使用写锁保护，确保线程安全
	initMu.Lock()
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
	initMu.Unlock()

	return nil
}

func Unsetup() error {
	return nil
}

// ensureInitialized 确保 logger 和 sugar 已初始化
// 如果未初始化，使用默认配置初始化
// 使用 sync.Once 确保线程安全的一次性初始化
// 如果 Setup() 已经初始化了 logger，则不会覆盖
func ensureInitialized() {
	// 快速路径：使用读锁检查是否已经初始化
	initMu.RLock()
	if sugar != nil {
		initMu.RUnlock()
		return
	}
	initMu.RUnlock()

	// 慢速路径：使用 sync.Once 确保只初始化一次
	initOnce.Do(func() {
		// 再次检查，防止 Setup() 在检查后、初始化前设置了 logger
		initMu.Lock()
		defer initMu.Unlock()

		if sugar != nil {
			// Setup() 已经初始化了，不需要再初始化
			return
		}

		// 使用默认配置初始化
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = customTimeEncoder{}.EncodeTime
		encoderConfig.EncodeLevel = customLevelEncoder{}.EncodeLevel
		encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
		encoderConfig.EncodeCaller = customCallerEncoder{}.EncodeCaller
		encoderConfig.ConsoleSeparator = " "

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zapcore.DebugLevel,
		)

		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		sugar = logger.Sugar()
	})
}

func Debug(args ...interface{}) {
	ensureInitialized()
	sugar.Debug(args...)
}

func Info(args ...interface{}) {
	ensureInitialized()
	sugar.Info(args...)
}

func Warn(args ...interface{}) {
	ensureInitialized()
	sugar.Warn(args...)
}

func Error(args ...interface{}) {
	ensureInitialized()
	sugar.Error(args...)
}

func Fatal(args ...interface{}) {
	ensureInitialized()
	sugar.Fatal(args...)
}

func Panic(args ...interface{}) {
	ensureInitialized()
	sugar.Panic(args...)
}

func Debugf(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Debugf(message, args...)
}

func Infof(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Infof(message, args...)
}

func Warnf(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Warnf(message, args...)
}

func Errorf(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Errorf(message, args...)
}

func Fatalf(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Fatalf(message, args...)
}

func Panicf(message string, args ...interface{}) {
	ensureInitialized()
	sugar.Panicf(message, args...)
}
