package fakedata

type Node struct {
	Type       string          `mapstructure:"type"`
	Items      []Node          `mapstructure:"items"`
	Properties map[string]Node `mapstructure:"properties"`
	Options    map[string]any  `mapstructure:"options"`
	Count      int             `mapstructure:"count"`
	Seed       uint64          `mapstructure:"seed"`
}
