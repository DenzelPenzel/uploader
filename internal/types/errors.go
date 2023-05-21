package types

import (
	"fmt"
)

// ErrFileNotExists is an error when image does not exist on storage
type ErrFileNotExists struct {
	ID ID
}

func (e ErrFileNotExists) Error() string {
	return fmt.Sprintf("No record found ID %v", e.ID)
}
