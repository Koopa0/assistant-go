package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// Load loads configuration from environment variables and YAML files
func Load() (*Config, error) {
	cfg := &Config{}

	// Set default values
	if err := setDefaults(cfg); err != nil {
		return nil, fmt.Errorf("failed to set default values: %w", err)
	}

	// Load from YAML file if exists
	if err := loadFromYAML(cfg); err != nil {
		return nil, fmt.Errorf("failed to load from YAML: %w", err)
	}

	// Override with environment variables
	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	// Validate configuration
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// loadFromYAML loads configuration from YAML files
func loadFromYAML(cfg *Config) error {
	// Determine config file path
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		// Try common locations
		candidates := []string{
			"configs/development.yaml",
			"configs/production.yaml",
			"config.yaml",
			".config.yaml",
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				configFile = candidate
				break
			}
		}
	}

	if configFile == "" {
		// No config file found, use defaults and environment variables only
		return nil
	}

	// Read and parse YAML file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse YAML config: %w", err)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(cfg *Config) error {
	return loadEnvRecursive(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem())
}

// loadEnvRecursive recursively loads environment variables into struct fields
func loadEnvRecursive(v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := loadEnvRecursive(field, fieldType.Type); err != nil {
				return err
			}
			continue
		}

		// Get environment variable name from tag
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// Get environment variable value
		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// Set field value based on type
		if err := setFieldValue(field, envValue); err != nil {
			return fmt.Errorf("failed to set field %s from env %s: %w", fieldType.Name, envTag, err)
		}
	}

	return nil
}

// setDefaults sets default values for configuration fields
func setDefaults(cfg *Config) error {
	return setDefaultsRecursive(reflect.ValueOf(cfg).Elem(), reflect.TypeOf(cfg).Elem())
}

// setDefaultsRecursive recursively sets default values for struct fields
func setDefaultsRecursive(v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := setDefaultsRecursive(field, fieldType.Type); err != nil {
				return err
			}
			continue
		}

		// Get default value from tag
		defaultTag := fieldType.Tag.Get("default")
		if defaultTag == "" {
			continue
		}

		// Set default value based on type
		if err := setFieldValue(field, defaultTag); err != nil {
			return fmt.Errorf("failed to set default for field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// setFieldValue sets a field value based on its type
func setFieldValue(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			// Handle time.Duration
			duration, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			field.SetInt(int64(duration))
		} else {
			// Handle regular integers
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid integer: %w", err)
			}
			field.SetInt(intValue)
		}
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean: %w", err)
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			// Handle string slices (comma-separated)
			values := strings.Split(value, ",")
			for i, v := range values {
				values[i] = strings.TrimSpace(v)
			}
			field.Set(reflect.ValueOf(values))
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() string {
	if configDir := os.Getenv("CONFIG_DIR"); configDir != "" {
		return configDir
	}

	// Default to configs directory
	return "configs"
}

// GetConfigFile returns the full path to the configuration file
func GetConfigFile(env string) string {
	configDir := GetConfigDir()
	return filepath.Join(configDir, fmt.Sprintf("%s.yaml", env))
}
