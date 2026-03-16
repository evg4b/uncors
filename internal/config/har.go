package config

// HARConfig defines settings for the HAR (HTTP Archive) collector middleware.
// When File is non-empty, all requests/responses passing through the proxy
// for this mapping will be recorded to the specified HAR file.
type HARConfig struct {
	File           string `mapstructure:"file"`
	CaptureCookies bool   `mapstructure:"capture-cookies"`
}

func (h HARConfig) Enabled() bool {
	return h.File != ""
}

func (h HARConfig) Clone() HARConfig {
	return HARConfig{
		File:           h.File,
		CaptureCookies: h.CaptureCookies,
	}
}
