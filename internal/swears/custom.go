package swears

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const customName = ".swears.json"

func LoadCustom(cwd, home string) ([]Entry, string, error) {
	for _, dir := range []string{cwd, home} {
		path := filepath.Join(dir, customName)
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, path, err
		}
		entries, err := ParseCustom(data)
		if err != nil {
			return nil, path, fmt.Errorf("%s: %w", path, err)
		}
		return entries, path, nil
	}
	return nil, "", nil
}

func ParseCustom(data []byte) ([]Entry, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil || items == nil {
		return nil, errors.New(`must be a JSON array of words or {"word": ..., "censored": ...} objects`)
	}
	entries := make([]Entry, 0, len(items))
	for i, item := range items {
		e, err := parseItem(item)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func parseItem(item json.RawMessage) (Entry, error) {
	var word string
	if err := json.Unmarshal(item, &word); err == nil && !bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
		if strings.TrimSpace(word) == "" {
			return Entry{}, errors.New("word is empty")
		}
		return Entry{Word: word, Suffix: true}, nil
	}
	var obj struct {
		Word     *string `json:"word"`
		Censored *string `json:"censored"`
		Prefix   *bool   `json:"prefix"`
		Suffix   *bool   `json:"suffix"`
	}
	dec := json.NewDecoder(bytes.NewReader(item))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&obj); err != nil || bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
		return Entry{}, errors.New(`must be a string or an object with "word" and optional "censored", "prefix", and "suffix"`)
	}
	if obj.Word == nil || strings.TrimSpace(*obj.Word) == "" {
		return Entry{}, errors.New(`"word" is missing or empty`)
	}
	e := Entry{Word: *obj.Word, Suffix: true}
	if obj.Prefix != nil {
		e.Prefix = *obj.Prefix
	}
	if obj.Suffix != nil {
		e.Suffix = *obj.Suffix
	}
	if obj.Censored != nil {
		if strings.TrimSpace(*obj.Censored) == "" {
			return Entry{}, errors.New(`"censored" is empty`)
		}
		e.Censored = *obj.Censored
	}
	return e, nil
}
