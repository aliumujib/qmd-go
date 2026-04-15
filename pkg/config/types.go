package config

type Collection struct {
	Path    string            `yaml:"path"`
	Pattern string            `yaml:"pattern"`
	Context map[string]string `yaml:"context,omitempty"`
	Update  string            `yaml:"update,omitempty"`
}

type Config struct {
	GlobalContext string                `yaml:"global_context,omitempty"`
	Collections   map[string]Collection `yaml:"collections"`
}
