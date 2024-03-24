package config

import (
	"firetunnel/util/slg"
	"log/slog"
)

var SlgOpts = slg.Options{
	Ansi:  true,
	Level: slog.LevelDebug,
}
