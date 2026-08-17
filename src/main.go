package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	IncludeDir string    `yaml:"include_dir"`
	Commands   []Command `yaml:"commands"`
	Command    *Command  `yaml:"command,omitempty"`
	Logging    Logging   `yaml:"logging"`
	PidFile    string    `yaml:"pid_file"`
	Timeout    string    `yaml:"default_timeout"`
}

type Command struct {
	Interval string   `yaml:"interval"`
	Path     string   `yaml:"path"`
	Args     []string `yaml:"args"`
	Timeout  string   `yaml:"timeout"`
}

type Logging struct {
	LogFile    string `yaml:"log_file"`
	LogLevel   string `yaml:"log_level"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	LocalTime  bool   `yaml:"local_time"`
	Compress   bool   `yaml:"compress"`
}

var cancelTasks context.CancelFunc

func parseDuration(interval string) (time.Duration, error) {
	if strings.Contains(interval, "d") {
		parts := strings.SplitN(interval, "d", 2)
		daysStr := parts[0]
		remaining := parts[1]

		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid day format: %v", err)
		}
		interval = fmt.Sprintf("%dh%s", days*24, remaining)
	}
	return time.ParseDuration(interval)
}

func loadConfigs(mainPath string) (*Config, error) {
	data, err := os.ReadFile(mainPath)
	if err != nil {
		return nil, err
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	if conf.IncludeDir != "" {
		files, _ := filepath.Glob(conf.IncludeDir)
		for _, f := range files {
			if filepath.Clean(f) == filepath.Clean(mainPath) {
				continue
			}

			extraData, err := os.ReadFile(f)
			if err != nil {
				continue
			}

			var extraConf Config
			if err := yaml.Unmarshal(extraData, &extraConf); err == nil {
				if len(extraConf.Commands) > 0 {
					conf.Commands = append(conf.Commands, extraConf.Commands...)
				}
				if extraConf.Command != nil {
					conf.Commands = append(conf.Commands, *extraConf.Command)
				}
			}
		}
	}
	return &conf, nil
}

func runCommand(ctx context.Context, path string, args []string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Msgf("Panic in runCommand: %v", r)
		}
	}()

	if !filepath.IsAbs(path) {
		log.Error().Str("path", path).Msg("Security Risk: Only absolute paths are allowed for commands")
		return
	}

	// nosemgrep
	cmd := exec.CommandContext(ctx, path, args...)
	output, err := cmd.CombinedOutput()

	fullCmd := fmt.Sprintf("%s %s", path, strings.Join(args, " "))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Error().Str("command", fullCmd).Msg("Command timed out")
		} else {
			log.Error().Err(err).Str("command", fullCmd).Msg("Execution failed")
		}
		return
	}
	log.Debug().Str("command", fullCmd).Str("output", string(output)).Msg("Success")
}

func scheduleCommand(ctx context.Context, cmdCfg Command, defaultTimeout time.Duration) {
	interval, err := parseDuration(cmdCfg.Interval)
	if err != nil {
		log.Error().Err(err).Str("interval", cmdCfg.Interval).Msg("Invalid interval, skipping task")
		return
	}
	if interval < time.Second {
		interval = time.Second
	}

	timeout := defaultTimeout
	if cmdCfg.Timeout != "" {
		if t, err := time.ParseDuration(cmdCfg.Timeout); err == nil {
			timeout = t
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// The parent context was cancelled (reload or shutdown)
			return
		case <-ticker.C:
			runCtx, cancel := context.WithTimeout(ctx, timeout)
			runCommand(runCtx, cmdCfg.Path, cmdCfg.Args)
			cancel()
		}
	}
}

func startTasks(configFile string) {
	if cancelTasks != nil {
		log.Info().Msg("Stopping existing tasks for reload...")
		cancelTasks()
	}

	config, err := loadConfigs(configFile)
	if err != nil {
		log.Error().Err(err).Msg("Configuration could not be loaded")
		return
	}

	setupLogging(*config)

	var ctx context.Context
	ctx, cancelTasks = context.WithCancel(context.Background())

	defaultTimeout, _ := time.ParseDuration(config.Timeout)
	if defaultTimeout == 0 {
		defaultTimeout = 10 * time.Minute
	}

	log.Info().Int("tasks", len(config.Commands)).Msg("Scheduling tasks")
	for _, cmd := range config.Commands {
		if cmd.Path == "" {
			continue
		}
		go scheduleCommand(ctx, cmd, defaultTimeout)
	}
}

func setupLogging(config Config) {
	l := config.Logging

	if l.LogFile != "" {
		log.Logger = zerolog.New(&lumberjack.Logger{
			Filename:   l.LogFile,
			MaxSize:    l.MaxSize,
			MaxBackups: l.MaxBackups,
			MaxAge:     l.MaxAge,
			Compress:   l.Compress,
		}).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	level := zerolog.InfoLevel
	if strings.ToLower(l.LogLevel) == "debug" {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)
}

func main() {
	var configFile string
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--config=") {
			configFile = strings.TrimPrefix(arg, "--config=")
		}
	}

	if configFile == "" {
		fmt.Println("Usage: edgestream-scheduler --config=<path>")
		os.Exit(1)
	}

	startTasks(configFile)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	log.Info().Msg("EdgeStream Scheduler event loop started")

	for {
		sig := <-sigChan
		switch sig {
		case syscall.SIGHUP:
			log.Info().Msg("SIGHUP received: Reloading configurations...")
			startTasks(configFile)

		case syscall.SIGTERM, syscall.SIGINT:
			log.Info().Msg("Shutting down EdgeStream Scheduler...")
			if cancelTasks != nil {
				cancelTasks()
			}
			// Give active processes a tiny window to exit cleanly
			time.Sleep(500 * time.Millisecond)
			return
		}
	}
}
