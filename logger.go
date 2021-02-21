package revel

import (
	"fmt"
	"strconv"

	"github.com/revel/revel/logger"
)

type compatLogger struct {
	println func(args ...interface{})
	printf  func(msg string, args ...interface{})
}

func (l compatLogger) Println(args ...interface{}) {
	l.println(args...)
}

func (l compatLogger) Print(args ...interface{}) {
	l.println(args...)
}

func (l compatLogger) Printf(msg string, args ...interface{}) {
	l.printf(msg, args...)
}

func toPrintln(f func(string, ...interface{})) func(args ...interface{}) {
	return func(args ...interface{}) {
		switch len(args) {
		case 0:
			f("")
		case 1:
			f(fmt.Sprint(args[0]))
		default:
			ctx := make([]interface{}, 0, 2*(len(args)-1))
			for idx, arg := range args[1:] {
				ctx[2*idx] = "arg" + strconv.Itoa(idx)
				ctx[2*idx+1] = arg
			}
			f(fmt.Sprint(args[0]), ctx...)
		}
	}
}

//Logger
var (
	// The root log is what all other logs are branched from, meaning if you set the handler for the root
	// it will adjust all children
	RootLog = logger.New()
	// This logger is the application logger, use this for your application log messages - ie jobs and startup,
	// Use Controller.Log for Controller logging
	// The requests are logged to this logger with the context of `section:requestlog`
	AppLog = RootLog.New("module", "app")
	// This is the logger revel writes to, added log messages will have a context of module:revel in them
	// It is based off of `RootLog`
	RevelLog = RootLog.New("module", "revel")

	// This is the handler for the AppLog, it is stored so that if the AppLog is changed it can be assigned to the
	// new AppLog
	appLogHandler *logger.CompositeMultiHandler

	// This oldLog is the revel logger, historical for revel, The application should use the AppLog or the Controller.oldLog
	// DEPRECATED
	oldLog = AppLog.New("section", "deprecated")
	// System logger
	SysLog = AppLog.New("section", "system")

	INFO = compatLogger{
		println: toPrintln(RevelLog.Info),
		printf:  RevelLog.Infof,
	}
	WARN = compatLogger{
		println: toPrintln(RevelLog.Warn),
		printf:  RevelLog.Warnf,
	}
	ERROR = compatLogger{
		println: toPrintln(RevelLog.Error),
		printf:  RevelLog.Errorf,
	}
)

// Initialize the loggers first
func init() {

	//RootLog.SetHandler(
	//	logger.LevelHandler(logger.LogLevel(log15.LvlDebug),
	//		logger.StreamHandler(os.Stdout, logger.TerminalFormatHandler(false, true))))
	initLoggers()
	OnAppStart(initLoggers, -5)

}
func initLoggers() {
	appHandle := logger.InitializeFromConfig(BasePath, Config)

	// Set all the log handlers
	setAppLog(AppLog, appHandle)
}

// Set the application log and handler, if handler is nil it will
// use the same handler used to configure the application log before
func setAppLog(appLog logger.MultiLogger, appHandler *logger.CompositeMultiHandler) {
	if appLog != nil {
		AppLog = appLog
	}

	if appHandler != nil {
		appLogHandler = appHandler
		// Set the app log and the handler for all forked loggers
		RootLog.SetHandler(appLogHandler)

		// Set the system log handler - this sets golang writer stream to the
		// sysLog router
		logger.SetDefaultLog(SysLog)
		SysLog.SetStackDepth(5)
		SysLog.SetHandler(appLogHandler)
	}
}
