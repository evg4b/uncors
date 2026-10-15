package commands

import (
	"fmt"

	"github.com/evg4b/uncors/internal/certs"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
	"github.com/spf13/pflag"
)

const defaultValidityDays = 365

type GenerateCertsCommand struct {
	validityDays int
	force        bool
	outputDir    string
	fs           afero.Fs
	output       contracts.Output
}

func NewGenerateCertsCommand(options ...Option) *GenerateCertsCommand {
	return helpers.ApplyOptions(&GenerateCertsCommand{}, options)
}

func (c *GenerateCertsCommand) DefineFlags(flags *pflag.FlagSet, version string) {
	flags.Usage = func() { tui.PrintUsage(flags, version) }
	flags.IntVar(&c.validityDays, "validity-days", defaultValidityDays, "Certificate validity period in days")
	flags.BoolVar(&c.force, "force", false, "Force overwrite existing CA certificates")
}

func (c *GenerateCertsCommand) Execute() error {
	outputDir, err := certs.Dir()
	if err != nil {
		return err
	}

	c.outputDir = outputDir
	certPath, keyPath := certs.Paths(c.outputDir)

	if !c.force {
		_, err := c.fs.Stat(certPath)
		if err == nil {
			c.output.Printf(
				fmt.Sprintf("CA certificate already exists at %s", certPath),
				"Use --force to overwrite",
			)

			return ErrCAAlreadyExists
		}

		_, err = c.fs.Stat(keyPath)
		if err == nil {
			c.output.Printf(
				fmt.Sprintf("CA private key already exists at %s", keyPath),
				"Use --force to overwrite",
			)

			return ErrCAKeyAlreadyExists
		}
	}

	c.output.Info("Generating CA certificate...")

	certPath, keyPath, err = certs.Generate(c.fs, c.outputDir, c.validityDays)
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %w", err)
	}

	c.output.InfoBox(
		"CA certificate generated successfully!",
		fmt.Sprintf("  Certificate: %s", certPath),
		fmt.Sprintf("  Private key: %s", keyPath),
		fmt.Sprintf("  Validity: %d days", c.validityDays),
		"",
		"To use auto-generated certificates:",
		"  1. Add the CA certificate to your system's trusted certificates",
		"  2. Configure HTTPS mappings in your uncors config without cert-file/key-file",
		"  3. UNCORS will automatically generate and sign certificates on-the-fly",
	)

	return nil
}
