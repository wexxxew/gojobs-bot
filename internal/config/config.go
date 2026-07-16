package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MaxPerRun  int    `yaml:"max_per_run"`
	MaxAgeDays int    `yaml:"max_age_days"`
	SeenFile   string `yaml:"seen_file"`

	Filter struct {
		RequireInclude bool     `yaml:"require_include"`
		Include        []string `yaml:"include"`
		Exclude        []string `yaml:"exclude"`
	} `yaml:"filter"`

	Sources struct {
		RemoteOK struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"remoteok"`
		JustJoin struct {
			Enabled    bool     `yaml:"enabled"`
			Experience []string `yaml:"experience"`
		} `yaml:"justjoin"`
		GolangProjects struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"golangprojects"`
		Habr struct {
			Enabled bool   `yaml:"enabled"`
			Query   string `yaml:"query"`
		} `yaml:"habr"`
		TGChannels []TGChannel `yaml:"tg_channels"`
	} `yaml:"sources"`
}

type TGChannel struct {
	Channel  string   `yaml:"channel"`
	Keywords []string `yaml:"keywords"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.MaxPerRun <= 0 {
		cfg.MaxPerRun = 15
	}
	if cfg.MaxAgeDays <= 0 {
		cfg.MaxAgeDays = 14
	}
	if cfg.SeenFile == "" {
		cfg.SeenFile = "seen.json"
	}
	return cfg, nil
}
