package har

import "os"

const (
	journalFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	archiveFlags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
)
