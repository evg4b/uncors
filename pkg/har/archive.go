package har

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const Version = "1.2"

var byteOrderMark = []byte{0xEF, 0xBB, 0xBF}

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

	buffered := bufio.NewReader(reader)

	prefix, err := buffered.Peek(len(byteOrderMark))
	if err == nil && bytes.Equal(prefix, byteOrderMark) {
		_, _ = buffered.Discard(len(byteOrderMark))
	}

	err = json.NewDecoder(buffered).Decode(&archive)
	if err != nil {
		return HAR{}, fmt.Errorf("har: cannot decode archive: %w", err)
	}

	return archive, nil
}
