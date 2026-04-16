package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ReceiverVarName string `mapstructure:"receiver_var_name"`
	ErrorVarName    string `mapstructure:"error_var_name"`
	ResultVarName   string `mapstructure:"result_var_name"`
	UseTestify      bool   `mapstructure:"use_testify"`
	UseRequire      bool   `mapstructure:"use_require"`
	CheckTypePrefix string `mapstructure:"check_type_prefix"`
	CheckTypeSuffix string `mapstructure:"check_type_suffix"`
	MockPrefix      string `mapstructure:"mock_prefix"`
	GenerateMocks   bool   `mapstructure:"generate_mocks"`
	GenerateChecks  bool   `mapstructure:"generate_checks"`
	AddTODOCases    bool   `mapstructure:"add_todo_cases"`
	NumberOfTODOs   int    `mapstructure:"number_of_todos"`
	TestStyle       string `mapstructure:"test_style"` // "check" | "table" | "simple"; empty = "check"
}

func DefaultConfig() *Config {
	return &Config{
		ReceiverVarName: "",
		ErrorVarName:    "err",
		ResultVarName:   "r",
		UseTestify:      true,
		UseRequire:      false,
		CheckTypePrefix: "",
		CheckTypeSuffix: "CheckFn",
		MockPrefix:      "mock",
		GenerateMocks:   true,
		GenerateChecks:  true,
		AddTODOCases:    true,
		NumberOfTODOs:   2,
	}
}

func Load(styleFile string) (*Config, error) {
	cfg := DefaultConfig()

	if styleFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = ""
		}

		searchPaths := []string{
			".go-testgen.yaml",
			".go-testgen.yml",
			".go-testgen.json",
		}

		if home != "" {
			searchPaths = append(searchPaths,
				filepath.Join(home, ".go-testgen.yaml"),
				filepath.Join(home, ".go-testgen.yml"),
				filepath.Join(home, ".go-testgen.json"),
			)
		}

		for _, p := range searchPaths {
			if _, err := os.Stat(p); err == nil {
				styleFile = p
				break
			}
		}
	}

	if styleFile == "" {
		return cfg, nil
	}

	ext := filepath.Ext(styleFile)
	if ext != "" {
		ext = ext[1:]
	}

	v := viper.New()
	v.SetConfigFile(styleFile)
	v.SetConfigType(ext)

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	err = v.Unmarshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}
