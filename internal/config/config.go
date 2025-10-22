package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the yamlcmt configuration
type Config struct {
	RepoOwner string         `yaml:"repo_owner"`
	RepoName  string         `yaml:"repo_name"`
	YAMLCmt   YAMLCmtConfig  `yaml:"yamlcmt"`
}

// YAMLCmtConfig represents the yamlcmt-specific configuration
type YAMLCmtConfig struct {
	Compare CompareConfig `yaml:"compare"`
}

// CompareConfig represents the compare command configuration
type CompareConfig struct {
	Template             string      `yaml:"template"`
	WhenHasAdditions     LabelConfig `yaml:"when_has_additions"`
	WhenHasDeletions     LabelConfig `yaml:"when_has_deletions"`
	WhenHasModifications LabelConfig `yaml:"when_has_modifications"`
	WhenNoChanges        LabelConfig `yaml:"when_no_changes"`
	DisableComment       bool        `yaml:"disable_comment"`
	DisableLabel         bool        `yaml:"disable_label"`
}

// LabelConfig represents label configuration
type LabelConfig struct {
	Label string `yaml:"label"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// GetRepoFullName returns the full repository name (owner/name)
func (c *Config) GetRepoFullName() string {
	if c.RepoOwner != "" && c.RepoName != "" {
		return c.RepoOwner + "/" + c.RepoName
	}
	return ""
}

// GetLabels returns all applicable labels based on diff result
// Labels are cumulative - if there are additions, deletions, and modifications,
// all three labels will be returned
func (c *CompareConfig) GetLabels(added, deleted, modified int) []string {
	var labels []string

	hasAdd := added > 0
	hasDelete := deleted > 0
	hasModify := modified > 0

	// No changes at all
	if !hasAdd && !hasDelete && !hasModify {
		if c.WhenNoChanges.Label != "" {
			labels = append(labels, c.WhenNoChanges.Label)
		}
		return labels
	}

	// Add label for additions
	if hasAdd && c.WhenHasAdditions.Label != "" {
		labels = append(labels, c.WhenHasAdditions.Label)
	}

	// Add label for deletions
	if hasDelete && c.WhenHasDeletions.Label != "" {
		labels = append(labels, c.WhenHasDeletions.Label)
	}

	// Add label for modifications
	if hasModify && c.WhenHasModifications.Label != "" {
		labels = append(labels, c.WhenHasModifications.Label)
	}

	return labels
}
