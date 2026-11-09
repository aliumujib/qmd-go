package config

type Collection struct {
	Path     string            `yaml:"path"`
	Pattern  string            `yaml:"pattern"`
	Patterns []string          `yaml:"patterns,omitempty"`  // when set, replaces Pattern
	Exclude  []string          `yaml:"exclude,omitempty"`   // doublestar globs, relative to Path
	MaxBytes int64             `yaml:"max_bytes,omitempty"` // 0 = no cap
	Context  map[string]string `yaml:"context,omitempty"`
	Update   string            `yaml:"update,omitempty"`
}

type Config struct {
	GlobalContext string                `yaml:"global_context,omitempty"`
	Collections   map[string]Collection `yaml:"collections"`
}
