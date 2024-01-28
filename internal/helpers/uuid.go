package helpers

import "github.com/google/uuid"

func GetUUID() string {
	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	return id.String()
}
