package server

import (
	"errors"
	"fmt"
	"github.com/denisschmidt/uploader/internal/types"
	"strings"
)

const (
	MULTI_PART_MAX_MEMORY = 1048576
	MAX_NOTE_LEN          = 500
	MAX_FILE_NAME_LEN     = 255
	RECORD_ID_LEN         = 10
)

var (
	ErrFilenameEmpty              = errors.New("filename cannot be empty")
	ErrFilenameTooLong            = errors.New("filename exceeds maximum length")
	ErrFilenameHasDotPrefix       = errors.New("filename has dot prefix")
	ErrFilenameIllegalCharacters  = errors.New("filename contains illegal characters")
	ErrFilenameEndsWithSpaceOrDot = errors.New("filename ends with space or dot")
	ErrFilenameIsWindowsReserved  = errors.New("filename is a reserved word on Windows")
)

func validateFileNote(s string) error {
	if s == "" {
		return nil
	}

	if len(s) > MAX_NOTE_LEN {
		return errors.New("text is too long")
	}

	if s == "null" || s == "undefined" {
		return errors.New("values of 'null' or 'undefined' are not allowed")
	}

	for _, tag := range []string{"<script>", "</script>", "<iframe>", "</iframe>"} {
		if strings.Contains(s, tag) {
			return errors.New("note must not contain HTML tags")
		}
	}

	return nil
}

var windowsReservedWords = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

func validateFilename(s string) error {
	if s == "" {
		return ErrFilenameEmpty
	}
	if len(s) > MAX_FILE_NAME_LEN {
		return ErrFilenameTooLong
	}
	if s == "." || strings.HasPrefix(s, "..") {
		return ErrFilenameHasDotPrefix
	}
	if strings.ContainsAny(s, "\\/\a\b\t\n\v\f\r\n") || strings.ContainsAny(s, "<>:\"|?*") {
		return ErrFilenameIllegalCharacters
	}
	if strings.HasSuffix(s, " ") || strings.HasSuffix(s, ".") {
		return ErrFilenameEndsWithSpaceOrDot
	}
	_, isWindowsReserved := windowsReservedWords[strings.ToUpper(s)]
	if isWindowsReserved {
		return ErrFilenameIsWindowsReserved
	}
	return nil
}

func getAllowCharsMapping() map[rune]bool {
	charsMapping := map[rune]bool{}
	for _, r := range []rune("abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789") {
		charsMapping[r] = true
	}
	return charsMapping

}

func parseRecordId(s string) (types.ID, error) {
	if len(s) != RECORD_ID_LEN {
		return types.ID(""), fmt.Errorf("ID (%s) has invalid length: got %d, want %d", s, len(s), RECORD_ID_LEN)
	}

	mp := getAllowCharsMapping()

	for _, c := range s {
		if _, ok := mp[c]; !ok {
			return types.ID(""), fmt.Errorf("wrong ID format (%s) unexpected char is %v", s, c)
		}
	}

	return types.ID(s), nil
}

func parseMetadataFromRequest(payload types.MetadataRequest) (types.Metadata, error) {
	err := validateFilename(payload.Filename)
	if err != nil {
		return types.Metadata{}, err
	}

	err = validateFileNote(payload.Note)
	if err != nil {
		return types.Metadata{}, err
	}

	return types.Metadata{
		Filename: types.Filename(payload.Filename),
		Note:     types.Note(payload.Note),
	}, nil

}
