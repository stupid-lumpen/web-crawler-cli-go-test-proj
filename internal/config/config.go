package config

import (
	"flag"
	"os"
	"time"
)

type Config struct {
	URLs       string
	ReqDepth   int
	Timeout    time.Duration
	ReqTimeout time.Duration
	OutputFile string
	LogFile    string
}

func Load() *Config {
	conf := Config{}

	fs := flag.NewFlagSet("crawler", flag.ExitOnError)

	fs.StringVar(
		&conf.URLs, "urls", "https://intechs.by", "Comma-seporated list of URLs to parse",
	)
	fs.IntVar(
		&conf.ReqDepth, "depth", 1, "The depth of recursice search in any page",
	)
	fs.DurationVar(
		&conf.Timeout, "timeout", time.Minute, "Max time for all program work",
	)
	fs.DurationVar(
		&conf.ReqTimeout, "request-timeout", time.Second*5,
		"Max time for handling one request",
	)
	fs.StringVar(
		&conf.OutputFile, "output", "result.json",
		"File where the result json tree is to be placed",
	)
	fs.StringVar(
		&conf.LogFile, "log", "result.log",
		"File where program work logs are to be placed",
	)

	_ = fs.Parse(os.Args[1:])

	return &conf
}
