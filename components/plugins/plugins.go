package plugins

import (
	"github.com/bocheninc/L0/components/utils"
)

// Plugin is the data of a plugin.
type Plugin struct {
	Name string
	Code []byte
}

// Make makes a Data by bytes.
func Make(data []byte) (*Plugin, error) {
	var plugin Plugin
	err := utils.Deserialize(data, &plugin)
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

// Bytes returns plugin's serialized data.
func (p *Plugin) Bytes() []byte {
	return utils.Serialize(p)
}
