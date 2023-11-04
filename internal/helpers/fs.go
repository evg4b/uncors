package helpers

import "github.com/spf13/afero"

func PassedOrOsFs(fs *afero.Fs) {
	if *fs == nil {
		*fs = afero.NewOsFs()
	}
}
