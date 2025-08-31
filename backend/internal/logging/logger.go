package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Fields map[string]interface{}

func output(level, msg string, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	fields["level"] = level
	fields["ts"] = time.Now().UTC().Format(time.RFC3339)
	fields["msg"] = msg
	b, err := json.Marshal(fields)
	if err != nil {
		// fallback to plain logging
		log.Printf("%s: %s (%v)\n", level, msg, fields)
		return
	}
	log.Println(string(b))
}

// Info logs an informational message with optional fields.
func Info(msg string, fields Fields) {
	output("info", msg, fields)
}

// Error logs an error message and includes the error text in the fields.
func Error(msg string, err error, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	output("error", msg, fields)
}

// Fatal logs a fatal error and exits the process.
func Fatal(msg string, err error, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	output("fatal", msg, fields)
	os.Exit(1)
}
