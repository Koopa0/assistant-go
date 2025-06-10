// Package config is responsible for loading, validating, and providing
// access to application configuration settings. It supports configuration
// from YAML files (e.g., development.yaml, production.yaml) and environment
// variables, with environment variables typically overriding file settings.
//
// The main 'Config' struct holds all configuration parameters, categorized
// into sections like Server, Database, AI providers, etc.
package config
