package har

import (
	"encoding/json"
	"fmt"
	"io"
)

const Version = "1.2"

func New(creator Creator, entries ...Entry) HAR {
	if entries == nil {
		entries = []Entry{}
	}

	return HAR{
		Log: Log{
			Version: Version,
			Creator: creator,
			Entries: entries,
		},
	}
}

func Encode(writer io.Writer, archive HAR) error {
	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return fmt.Errorf("har: cannot encode archive: %w", err)
	}

	_, err = writer.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("har: cannot write archive: %w", err)
	}

	return nil
}

func Decode(reader io.Reader) (HAR, error) {
	var archive HAR

	err := json.NewDecoder(reader).Decode(&archive)
	if err != nil {
		return HAR{}, fmt.Errorf("har: cannot decode archive: %w", err)
	}

	return archive, nil
}
