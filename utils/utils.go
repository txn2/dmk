package utils

import (
	"github.com/satori/go.uuid"
)

type Utils struct {
}

func (u Utils) GetUuidString() string {
	return uuid.NewV4().String()
}
