package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	infratls "github.com/evg4b/uncors/internal/infra/tls"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const (
	defaultValidityDays = 365
	defaultConfigDir    = ".config/uncors"
)

// GenerateCertsCommand handles the 'generate-certs' command.
type GenerateCertsCommand struct {
	validityDays int
	force        bool
	outputDir    string
	fs           afero.Fs
}

// NewGenerateCertsCommand creates a new generate-certs command.
func NewGenerateCertsCommand(fs afero.Fs) *GenerateCertsCommand {
	return &GenerateCertsCommand{
		fs: fs,
	}
}

// DefineFlags defines command-line flags for the generate-certs command.
func (c *GenerateCertsCommand) DefineFlags(flags *pflag.FlagSet) {
	flags.IntVar(&c.validityDays, "validity-days", defaultValidityDays, "Certificate validity period in days")
	flags.BoolVar(&c.force, "force", false, "Force overwrite existing CA certificates")
}

// Execute runs the generate-certs command.
func (c *GenerateCertsCommand) Execute() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	c.outputDir = filepath.Join(homeDir, defaultConfigDir)

	certPath := filepath.Join(c.outputDir, infratls.CACertFileName)
	keyPath := filepath.Join(c.outputDir, infratls.CAKeyFileName)

	if !c.force {
		if _, err := c.fs.Stat(certPath); err == nil {
			log.Errorf("CA certificate already exists at %s", certPath)
			log.Info("Use --force to overwrite")

			return ErrCAAlreadyExists
		}
		if _, err := c.fs.Stat(keyPath); err == nil {
			log.Errorf("CA private key already exists at %s", keyPath)
			log.Info("Use --force to overwrite")

			return ErrCAKeyAlreadyExists
		}
	}

	log.Info("Generating CA certificate...")
	certPath, keyPath, err = infratls.GenerateCA(infratls.CAConfig{
		ValidityDays: c.validityDays,
		OutputDir:    c.outputDir,
		Fs:           c.fs,
	})
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	log.Infof("CA certificate generated successfully!")
	log.Infof("  Certificate: %s", certPath)
	log.Infof("  Private key: %s", keyPath)
	log.Infof("  Validity: %d days", c.validityDays)
	log.Info("")
	log.Info("To use auto-generated certificates:")
	log.Info("  1. Add the CA certificate to your system's trusted certificates")
	log.Info("  2. Configure HTTPS mappings in your uncors config without cert-file/key-file")
	log.Info("  3. UNCORS will automatically generate and sign certificates on-the-fly")

	return nil
}
