package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"os"
)

type ConfigLoader struct {
	viper  *viper.Viper
	fs     afero.Fs
	config chan *config.UncorsConfig
}

func NewConfigLoader(viper *viper.Viper, fs afero.Fs) *ConfigLoader {
	return &ConfigLoader{
		viper:  viper,
		fs:     fs,
		config: make(chan *config.UncorsConfig),
	}
}

func (c *ConfigLoader) Init() (mesage tea.Msg) {
	viper.OnConfigChange(func(in fsnotify.Event) {
		c.config <- loadConfiguration(c.viper, c.fs)
	})
	viper.WatchConfig()

	return c.Tick()
}

func (c *ConfigLoader) Tick() tea.Msg {
	return <-c.config
}

func (c *ConfigLoader) Load() *config.UncorsConfig {
	return loadConfiguration(c.viper, c.fs)
}

func loadConfiguration(viperInstance *viper.Viper, fs afero.Fs) *config.UncorsConfig {
	uncorsConfig := config.LoadConfiguration(viperInstance, os.Args)
	err := validators.ValidateConfig(uncorsConfig, fs)
	if err != nil {
		panic(err)
	}

	if uncorsConfig.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Enabled debug messages")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	return uncorsConfig
}
