// internal/logger/logger.go
package logger

import (
    "io"
    "os"
    "time"
    
    "github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init(level string, output string) {
    var w io.Writer
    
    switch output {
    case "stdout":
        w = os.Stdout
    default:
        file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
        if err != nil {
            panic(err)
        }
        w = file
    }
    
    lvl, err := zerolog.ParseLevel(level)
    if err != nil {
        lvl = zerolog.InfoLevel
    }
    
    Log = zerolog.New(w).
        Level(lvl).
        With().
        Timestamp().
        Caller().
        Logger()
    
    zerolog.TimeFieldFormat = time.RFC3339
}

func Debug() *zerolog.Event { return Log.Debug() }
func Info() *zerolog.Event  { return Log.Info() }
func Warn() *zerolog.Event  { return Log.Warn() }
func Error() *zerolog.Event { return Log.Error() }
func Fatal() *zerolog.Event { return Log.Fatal() }