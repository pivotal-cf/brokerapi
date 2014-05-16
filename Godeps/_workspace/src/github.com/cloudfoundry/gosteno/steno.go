package gosteno

import (
	"encoding/json"
	"sync"
)

// Global configs
var config Config

// loggersMutex protects accesses to loggers and regexp
var loggersMutex sync.Mutex

// loggers only saves BaseLogger
var loggers = make(map[string]*BaseLogger)

func Init(c *Config) {
	config = *c

	if config.Level == (LogLevel{}) {
		config.Level = LOG_INFO
	}
	if config.Codec == nil {
		config.Codec = NewJsonCodec()
	}
	if config.Sinks == nil {
		config.Sinks = []Sink{}
	}

	for _, sink := range config.Sinks {
		if sink.GetCodec() == nil {
			sink.SetCodec(config.Codec)
		}
	}

	for name, _ := range loggers {
		loggers[name] = nil
	}
}

func NewLogger(name string) *Logger {
	loggersMutex.Lock()
	defer loggersMutex.Unlock()

	l := loggers[name]
	if l == nil {
		bl := &BaseLogger{
			name:  name,
			sinks: config.Sinks,
			level: computeLevel(name),
		}

		loggers[name] = bl
		l = bl
	}

	return &Logger{L: l}
}

func loggersInJson() string {
	bytes, _ := json.Marshal(loggers)
	return string(bytes)
}
