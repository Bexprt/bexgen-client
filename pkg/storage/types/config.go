package types

type FactoryConfig struct {
	Driver     string         `yaml:"driver"`
	ConfigPath string         `yaml:"-"`
	Options    map[string]any `yaml:"options"`
}

type RootYAML struct {
	Storage FactoryConfig `yaml:"storage"`
}
