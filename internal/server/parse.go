package server

import (
	"errors"
	"github.com/denisschmidt/uploader/internal/types"
)

const (
	MULTI_PART_MAX_MEMORY = 1048576
	MAX_NOTE_LEN          = 512
)

func parseFileNote(s string) (types.Note, error) {
	if s == "" {
		return types.Note{}, nil
	}

	if len(s) > MAX_NOTE_LEN {
		return types.Note{}, errors.New("text is too long")
	}

	return types.Note{Value: &s}, nil
}
