package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"

	"github.com/rupor-github/gencfg"

	"sync2kindle/thumbs"
)

//go:embed config.yaml.tmpl
var ConfigTmpl []byte

type Config struct {
	SourcePath   string `yaml:"source" sanitize:"path_abs,path_toslash" validate:"required,dir"`
	TargetPath   string `yaml:"target" sanitize:"path_clean,path_toslash" validate:"required,gt=0"`
	HistoryPath  string `yaml:"history" sanitize:"path_clean,assure_dir_exists" validate:"required,dir"`
	DeviceSerial string `yaml:"device_serial" validate:"omitempty,gt=0"`

	BookExtensions  []string `yaml:"book_extensions" validate:"required,gt=0"`
	ThumbExtensions []string `yaml:"thumb_extensions" validate:"required,gt=0"`

	Thumbnails thumbs.ThumbnailsConfig `yaml:"thumbnails"`

	Logging   LoggingConfig  `yaml:"logging"`
	Reporting ReporterConfig `yaml:"reporting"`
}

func unmarshalConfig(data []byte, cfg *Config, process bool) (*Config, error) {
	// We want to use only fields we defined so we cannot use yaml.Unmarshal directly here
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("failed to decode configuration data: %w", err)
	}
	if process {
		// sanitize and validate what has been loaded
		if err := gencfg.Sanitize(cfg); err != nil {
			return nil, err
		}
		if err := gencfg.Validate(cfg); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// LoadConfiguration reads the configuration from the file at the given path, superimposes its values on
// top of expanded configuration tamplate to provide sane defaults and performs validation.
func LoadConfiguration(path string) (*Config, error) {
	haveFile := len(path) > 0

	data, err := gencfg.Process(ConfigTmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to process configuration template: %w", err)
	}
	cfg, err := unmarshalConfig(data, &Config{}, !haveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to process configuration template: %w", err)
	}
	if !haveFile {
		return cfg, nil
	}

	// overwrite cfg values with values from the file
	data, err = os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	cfg, err = unmarshalConfig(data, cfg, haveFile)
	if err != nil {
		return nil, fmt.Errorf("failed to process configuration file: %w", err)
	}
	return cfg, nil
}

// Prepare generates configuration file from template and returns it as a byte slice.
func Prepare() ([]byte, error) {
	return gencfg.Process(ConfigTmpl)
}

func Dump(cfg *Config) ([]byte, error) {
	data, err := yaml.Marshal(*cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config to yaml: %v", err)
	}
	return data, nil
}
