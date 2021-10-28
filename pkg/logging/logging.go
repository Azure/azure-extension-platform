package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/Azure/azure-extension-platform/pkg/handlerenv"
)

const (
	logLevelError   = "Error "
	logLevelWarning = "Warning "
	logLevelInfo    = "Info "
)

// ExtensionLogger exposes logging capabilities to the extension
// It automatically appends time stamps and debug level to each message
// and ensures all logs are placed in the logs folder passed by the agent
// TODO: eventually we need to support cycling of logs to prevent filling up the disk
type ExtensionLogger struct {
	errorLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	file        *os.File
}

// New creates a new logging instance. If the handlerEnvironment is nil, we'll use a
// standard output logger
func New(he *handlerenv.HandlerEnvironment) *ExtensionLogger {
	if he == nil {
		return newStandardOutput()
	}

	fileName := fmt.Sprintf("log_%v", strconv.FormatInt(time.Now().UTC().Unix(), 10))
	filePath := path.Join(he.LogFolder, fileName)
	writer, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return newStandardOutput()
	}

	return &ExtensionLogger{
		errorLogger: log.New(writer, logLevelError, log.Ldate|log.Ltime|log.LUTC),
		infoLogger:  log.New(writer, logLevelInfo, log.Ldate|log.Ltime|log.LUTC),
		warnLogger:  log.New(writer, logLevelWarning, log.Ldate|log.Ltime|log.LUTC),
		file:        writer,
	}
}

func GetCallStack() string {
	return string(debug.Stack())
}

func newStandardOutput() *ExtensionLogger {
	return &ExtensionLogger{
		errorLogger: log.New(os.Stdout, logLevelError, 0),
		infoLogger:  log.New(os.Stdout, logLevelInfo, 0),
		warnLogger:  log.New(os.Stdout, logLevelWarning, 0),
		file:        nil,
	}
}

// Close closes the file
func (logger *ExtensionLogger) Close() {
	if logger.file != nil {
		logger.file.Close()
	}
}

// Error logs an error. Format is the same as fmt.Print
func (logger *ExtensionLogger) Error(format string, v ...interface{}) {
	logger.errorLogger.Printf(format+"\n", v...)
	logger.errorLogger.Printf(GetCallStack() + "\n")
}

// Warn logs a warning. Format is the same as fmt.Print
func (logger *ExtensionLogger) Warn(format string, v ...interface{}) {
	logger.warnLogger.Printf(format+"\n", v...)
}

// Info logs an information statement. Format is the same as fmt.Print
func (logger *ExtensionLogger) Info(format string, v ...interface{}) {
	logger.infoLogger.Printf(format+"\n", v...)
}

// Error logs an error. Get the message from a stream directly
func (logger *ExtensionLogger) ErrorFromStream(prefix string, streamReader io.Reader) {
	logger.errorLogger.Print(prefix)
	io.Copy(logger.errorLogger.Writer(), streamReader)
	logger.errorLogger.Writer().Write([]byte(fmt.Sprintln())) // add a newline at the end of the stream contents
}

// Warn logs a warning. Get the message from a stream directly
func (logger *ExtensionLogger) WarnFromStream(prefix string, streamReader io.Reader) {
	logger.warnLogger.Print(prefix)
	io.Copy(logger.warnLogger.Writer(), streamReader)
	logger.warnLogger.Writer().Write([]byte(fmt.Sprintln())) // add a newline at the end of the stream contents
}

// Info logs an information statement. Get the message from a stream directly
func (logger *ExtensionLogger) InfoFromStream(prefix string, streamReader io.Reader) {
	logger.infoLogger.Print(prefix)
	io.Copy(logger.infoLogger.Writer(), streamReader)
	logger.infoLogger.Writer().Write([]byte(fmt.Sprintln())) // add a newline at the end of the stream contents
}
