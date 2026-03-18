package zap

type Logger struct{}

type SugaredLogger struct{}

type Field struct{}

func String(key, val string) Field { return Field{} }

func (*Logger) Info(msg string, fields ...Field)  {}
func (*Logger) Warn(msg string, fields ...Field)  {}
func (*Logger) Error(msg string, fields ...Field) {}
func (*Logger) Debug(msg string, fields ...Field) {}

func (*SugaredLogger) Info(msg string)                          {}
func (*SugaredLogger) Infof(msg string, args ...any)            {}
func (*SugaredLogger) Infow(msg string, keysAndValues ...any)   {}
func (*SugaredLogger) Warn(msg string)                          {}
func (*SugaredLogger) Warnf(msg string, args ...any)            {}
func (*SugaredLogger) Warnw(msg string, keysAndValues ...any)   {}
func (*SugaredLogger) Error(msg string)                         {}
func (*SugaredLogger) Errorf(msg string, args ...any)           {}
func (*SugaredLogger) Errorw(msg string, keysAndValues ...any)  {}
func (*SugaredLogger) Debug(msg string)                         {}
func (*SugaredLogger) Debugf(msg string, args ...any)           {}
func (*SugaredLogger) Debugw(msg string, keysAndValues ...any)  {}
func (*SugaredLogger) Fatal(msg string)                         {}
func (*SugaredLogger) Fatalf(msg string, args ...any)           {}
func (*SugaredLogger) Fatalw(msg string, keysAndValues ...any)  {}
func (*SugaredLogger) Panic(msg string)                         {}
func (*SugaredLogger) Panicf(msg string, args ...any)           {}
func (*SugaredLogger) Panicw(msg string, keysAndValues ...any)  {}
func (*SugaredLogger) DPanic(msg string)                        {}
func (*SugaredLogger) DPanicf(msg string, args ...any)          {}
func (*SugaredLogger) DPanicw(msg string, keysAndValues ...any) {}
