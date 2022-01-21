package log_test

import (
	"bytes"
	"testing"

	"github.com/TouchBistro/goutils/log"
)

func TestLogging(t *testing.T) {
	tests := []struct {
		name     string
		minLevel log.Level
		run      func(*log.Logger)
		want     string
	}{
		{
			name:     "Log",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Log(log.LevelInfo, "some log")
			},
			want: `level=info message="some log"` + "\n",
		},
		{
			name:     "Debug",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Debug("some log")
			},
			want: `level=debug message="some log"` + "\n",
		},
		{
			name:     "Info",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Info("some log")
			},
			want: `level=info message="some log"` + "\n",
		},
		{
			name:     "Warn",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Warn("some log")
			},
			want: `level=warn message="some log"` + "\n",
		},
		{
			name:     "Error",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Error("some log")
			},
			want: `level=error message="some log"` + "\n",
		},
		{
			name:     "Logf",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Logf(log.LevelInfo, "some log: %d", 123)
			},
			want: `level=info message="some log: 123"` + "\n",
		},
		{
			name:     "Debugf",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Debugf("some log: %d", 123)
			},
			want: `level=debug message="some log: 123"` + "\n",
		},
		{
			name:     "Infof",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Infof("some log: %d", 123)
			},
			want: `level=info message="some log: 123"` + "\n",
		},
		{
			name:     "Warnf",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Warnf("some log: %d", 123)
			},
			want: `level=warn message="some log: 123"` + "\n",
		},
		{
			name:     "Errorf",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.Errorf("some log: %d", 123)
			},
			want: `level=error message="some log: 123"` + "\n",
		},
		{
			name:     "WithFields",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.WithFields(log.Fields{"foo": "bar"}).Info("some log")
			},
			want: `level=info message="some log" foo=bar` + "\n",
		},
		{
			name:     "WithFields twice",
			minLevel: log.LevelDebug,
			run: func(l *log.Logger) {
				l.WithFields(log.Fields{"foo": "bar"}).WithFields(log.Fields{"baz": "qux"}).Infof("some log: %d", 123)
			},
			want: `level=info message="some log: 123" baz=qux foo=bar` + "\n",
		},
		{
			name:     "level disabled",
			minLevel: log.LevelInfo,
			run: func(l *log.Logger) {
				l.Debug("won't log")
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := log.New(
				log.WithOutput(&buf),
				log.WithFormatter(&log.TextFormatter{DisableTimestamp: true}),
				log.WithLevel(tt.minLevel),
			)
			tt.run(logger)
			if buf.String() != tt.want {
				t.Fatalf("\ngot log: %s\nwant: %s", buf.String(), tt.want)
			}
		})
	}
}

func TestLoggerHook(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(
		log.WithOutput(&buf),
		log.WithFormatter(&log.TextFormatter{DisableTimestamp: true}),
		log.WithLevel(log.LevelInfo),
	)
	logger.AddHook(testHook{})
	logger.WithFields(log.Fields{"foo": "bar"}).Warn("oops")

	want := "level=warn message=oops foo=bar hook_ran=true\n"
	if buf.String() != want {
		t.Fatalf("\ngot log: %s\nwant: %s", buf.String(), want)
	}
}

type testHook struct{}

func (h testHook) Run(e *log.Entry) error {
	e.Fields["hook_ran"] = true
	return nil
}

func BenchmarkNoFields(b *testing.B) {
	logger := log.New(
		log.WithOutput(&bytes.Buffer{}),
		log.WithFormatter(&log.TextFormatter{}),
		log.WithLevel(log.LevelInfo),
	)
	for i := 0; i < b.N; i++ {
		logger.Info("some log")
	}
}

func BenchmarkFewFields(b *testing.B) {
	logger := log.New(
		log.WithOutput(&bytes.Buffer{}),
		log.WithFormatter(&log.TextFormatter{}),
		log.WithLevel(log.LevelInfo),
	)
	for i := 0; i < b.N; i++ {
		logger.WithFields(log.Fields{
			"foo": "bar",
			"baz": "qux",
			"abc": 123,
		}).Info("some log")
	}
}

func BenchmarkManyFields(b *testing.B) {
	logger := log.New(
		log.WithOutput(&bytes.Buffer{}),
		log.WithFormatter(&log.TextFormatter{}),
		log.WithLevel(log.LevelInfo),
	)
	for i := 0; i < b.N; i++ {
		logger.WithFields(log.Fields{
			"foo":        "bar",
			"baz":        "qux",
			"abc":        123,
			"a":          "b",
			"c":          "d",
			"e":          "f",
			"g":          "h",
			"i":          "j",
			"k":          "l",
			"m":          "n",
			"o":          "p",
			"q":          "r",
			"s":          "t",
			"u":          "v",
			"w":          "x",
			"y":          "z",
			"nullary":    "0-ary",
			"unary":      "1-ary",
			"binary":     "2-ary",
			"ternary":    "3-ary",
			"quaternary": "4-ary",
			"quinary":    "5-ary",
			"senary":     "6-ary",
			"septenary":  "7-ary",
			"octonary":   "8-ary",
			"nonary":     "9-ary",
			"decenary":   "10-ary",
			"object": map[string]interface{}{
				"key":   "value",
				"empty": nil,
			},
		}).Info("some log")
	}
}
