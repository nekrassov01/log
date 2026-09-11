// Examples demonstrates the default and labeled styles with terminal-aware coloring.
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/nekrassov01/log"
)

func main() {
	var l *slog.Logger
	var h slog.Handler

	// Use the default style with function names and CLI execution details.
	h = log.NewCLIHandler(os.Stdout,
		log.WithTime(),
		log.WithLevel(slog.LevelDebug),
		log.WithSourceFunction(),
		log.WithLabel("DEFAULT STYLE:"),
		log.WithStyle(log.DefaultStyle()),
	)
	l = slog.New(h).
		WithGroup("default").
		With(
			slog.String("version", "1.0.0"),
			slog.String("command", "build"),
		)
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
	println()

	// Customize timestamp brackets and level backgrounds while preserving other defaults.
	h = log.NewCLIHandler(os.Stdout,
		log.WithTime(),
		log.WithTimeLayout(time.RFC1123Z),
		log.WithLevel(slog.LevelDebug),
		log.WithSourcePath(),
		log.WithLabel("LABELED STYLE:"),
		log.WithStyle(log.NewStyle(
			log.WithTimeStyle(log.TimeStyle{
				Prefix: log.AffixStyle{
					Text: "[",
				},
				Suffix: log.AffixStyle{
					Text: "]",
				},
			}),
			log.WithLevelStyle(map[slog.Level]log.LevelStyle{
				slog.LevelDebug: {
					Text:  "DBG",
					Color: log.NewColor(48, 2, 95, 95, 255),
					Width: 5,
				},
				slog.LevelInfo: {
					Text:  "INF",
					Color: log.NewColor(48, 2, 95, 255, 215),
					Width: 5,
				},
				slog.LevelWarn: {
					Text:  "WRN",
					Color: log.NewColor(48, 2, 215, 255, 135),
					Width: 5,
				},
				slog.LevelError: {
					Text:  "ERR",
					Color: log.NewColor(48, 2, 255, 95, 135),
					Width: 5,
				},
			}),
		)),
	)
	l = slog.New(h).
		WithGroup("labeled").
		With(
			slog.String("version", "1.0.0"),
			slog.String("command", "build"),
		)
	l.Debug("debug message")
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")
	println()
}
