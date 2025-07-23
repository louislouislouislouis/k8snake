package utils

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func configureLogger() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "3:04PM",
		NoColor:    false,
		FormatLevel: func(i any) string {
			switch i {
			case "debug":
				return "\x1b[36mDBG\x1b[0m" // cyan
			case "info":
				return "\x1b[32mINF\x1b[0m" // green
			case "warn":
				return "\x1b[33mWRN\x1b[0m" // yellow
			case "error":
				return "\x1b[31mERR\x1b[0m" // red
			default:
				return fmt.Sprintf("%s", i)
			}
		},
	})

	if os.Getenv("LOG_DEBUG") == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
