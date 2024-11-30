package fakedata

import (
	"github.com/evg4b/uncors/internal/helpers"
)

type Node struct {
	Type       string          `mapstructure:"type"`
	Item       *Node           `mapstructure:"item"`
	Properties map[string]Node `mapstructure:"properties"`
	Options    map[string]any  `mapstructure:"options"`
	Count      int             `mapstructure:"count"`
}

func (root *Node) Clone() *Node {
	return &Node{
		Type:       root.Type,
		Item:       root.Item,
		Properties: helpers.CloneMap(root.Properties),
		Options:    helpers.CloneMap(root.Options),
		Count:      root.Count,
	}
}
