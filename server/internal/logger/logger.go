package logger

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func New(level, format string) *zerolog.Logger {
	var output io.Writer = os.Stdout
	if strings.EqualFold(format, "text") {
		output = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	}
	log := zerolog.New(output).With().Timestamp().Logger()
	log = log.Level(parseLevel(level))
	return &log
}

func NewDiscard() *zerolog.Logger {
	log := zerolog.New(io.Discard)
	return &log
}

func parseLevel(value string) zerolog.Level {
	level, err := zerolog.ParseLevel(strings.ToLower(value))
	if err != nil {
		return zerolog.InfoLevel
	}
	return level
}
