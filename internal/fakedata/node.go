package fakedata

type Node struct {
	Type       string          `mapstructure:"type"`
	Item       *Node           `mapstructure:"item"`
	Properties map[string]Node `mapstructure:"properties"`
	Options    map[string]any  `mapstructure:"options"`
	Count      int             `mapstructure:"count"`
	Seed       uint64          `mapstructure:"seed"`
}
