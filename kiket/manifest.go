package kiket

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadManifest loads an extension manifest from file.
func LoadManifest(manifestPath string) (*Manifest, error) {
	paths := []string{manifestPath}
	if manifestPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		paths = []string{
			filepath.Join(cwd, "extension.yaml"),
			filepath.Join(cwd, "manifest.yaml"),
			filepath.Join(cwd, "extension.yml"),
			filepath.Join(cwd, "manifest.yml"),
		}
	}

	for _, p := range paths {
		if p == "" {
			continue
		}

		content, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		var manifest Manifest
		if err := yaml.Unmarshal(content, &manifest); err != nil {
			continue
		}

		return &manifest, nil
	}

	return nil, nil
}

// SettingsDefaults extracts default values from a manifest.
func SettingsDefaults(manifest *Manifest) Settings {
	if manifest == nil || len(manifest.Settings) == 0 {
		return Settings{}
	}

	defaults := make(Settings)
	for _, setting := range manifest.Settings {
		if setting.Default != nil {
			defaults[setting.Key] = setting.Default
		}
	}

	return defaults
}

// SecretKeys extracts secret keys from a manifest.
func SecretKeys(manifest *Manifest) []string {
	if manifest == nil || len(manifest.Settings) == 0 {
		return nil
	}

	var keys []string
	for _, setting := range manifest.Settings {
		if setting.Secret {
			keys = append(keys, setting.Key)
		}
	}

	return keys
}

// ApplySecretEnvOverrides applies KIKET_SECRET_* environment variable overrides.
func ApplySecretEnvOverrides(settings Settings, secrets []string) Settings {
	updated := make(Settings)
	for k, v := range settings {
		updated[k] = v
	}

	for _, key := range secrets {
		envKey := "KIKET_SECRET_" + toUpperSnake(key)
		if envValue := os.Getenv(envKey); envValue != "" {
			updated[key] = envValue
		}
	}

	return updated
}

func toUpperSnake(s string) string {
	result := make([]byte, 0, len(s)*2)
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c))
		} else if c >= 'a' && c <= 'z' {
			result = append(result, byte(c-32))
		} else if c == '-' || c == '.' {
			result = append(result, '_')
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
