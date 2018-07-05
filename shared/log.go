package shared

import (
	"fmt"
	"runtime"
	"strings"
)

type Logger struct {
	trace, debug, err bool
}

func NewLogger(t, d, e bool) *Logger {
	return &Logger{t, d, e}
}

// e.g. Trace(":8090", 12345)
func (l *Logger) Trace(args ...interface{}) {
	if l.trace {
		funcName, file, line := getCallerInfo()
		str := fmt.Sprintf("<T> [%s:%d] %s -", file, line, funcName)
		msg := append([]interface{}{str}, args...)
		fmt.Println(msg...)
	}
}

// e.g. Debug("client %s connected after %d seconds", "numberOne", 10)
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.debug {
		funcName, file, line := getCallerInfo()
		str := fmt.Sprintf("<D> [%s:%d] %s - %s\n", file, line, funcName, msg)
		fmt.Printf(str, args...)
	}
}

// e.g. Error("server failed to start [%s]", err.Error())
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.err {
		funcName, file, line := getCallerInfo()
		str := fmt.Sprintf("<E> [%s:%d] %s - %s\n", file, line, funcName, msg)
		fmt.Printf(str, args...)
	}
}

func getCallerInfo() (funcName, file string, line int) {
	pc, file, line, ok := runtime.Caller(2)
	if ok {
		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]

		funcName = runtime.FuncForPC(pc).Name()
		parts = strings.Split(funcName, ".")
		funcName = parts[len(parts)-1]
	}
	return
}
