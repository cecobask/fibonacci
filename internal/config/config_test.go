package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/cecobask/fibonacci/internal/config"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		env     map[string]string
		want    config.Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: config.Default(),
		},
		{
			name: "all set",
			env: map[string]string{
				"PORT":             "9090",
				"LOG_LEVEL":        "debug",
				"LOG_FORMAT":       "text",
				"SHUTDOWN_TIMEOUT": "30s",
				"SHUTDOWN_DELAY":   "5s",
			},
			want: config.Config{
				Port:            9090,
				LogLevel:        slog.LevelDebug,
				LogFormat:       config.LogFormatText,
				ShutdownTimeout: 30 * time.Second,
				ShutdownDelay:   5 * time.Second,
			},
		},
		{name: "invalid port", env: map[string]string{"PORT": "http"}, wantErr: true},
		{name: "port out of range", env: map[string]string{"PORT": "70000"}, wantErr: true},
		{name: "invalid log level", env: map[string]string{"LOG_LEVEL": "verbose"}, wantErr: true},
		{name: "invalid log format", env: map[string]string{"LOG_FORMAT": "xml"}, wantErr: true},
		{name: "invalid shutdown timeout", env: map[string]string{"SHUTDOWN_TIMEOUT": "soon"}, wantErr: true},
		{name: "negative shutdown delay", env: map[string]string{"SHUTDOWN_DELAY": "-1s"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.Load(func(key string) string { return tt.env[key] })
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
