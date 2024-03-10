package helpers

import "github.com/google/uuid"

func GetUUID() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	return id
}
