// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package fsgraph

import (
	fmt "fmt"
	io "io"
	strconv "strconv"
)

// a generic file
type File interface {
	IsFile()
}

// the contents of a file
type FileContents struct {
	Data     string   `json:"data"`
	Next     *Int64   `json:"next"`
	Encoding Encoding `json:"encoding"`
	Warning  *string  `json:"warning"`
}

// a representation of the file's mode
type FileMode struct {
	Type   FileType `json:"type"`
	Perm   int      `json:"perm"`
	Sticky bool     `json:"sticky"`
}

type OKResult struct {
	S       string  `json:"s"`
	Warning *string `json:"warning"`
}

func (OKResult) IsResult() {}

// a generic result of an operation
type Result interface {
	IsResult()
}

// file contents (read) or write encoding
type Encoding string

const (
	EncodingAuto   Encoding = "auto"
	EncodingUtf8   Encoding = "utf8"
	EncodingBase64 Encoding = "base64"
)

func (e Encoding) IsValid() bool {
	switch e {
	case EncodingAuto, EncodingUtf8, EncodingBase64:
		return true
	}
	return false
}

func (e Encoding) String() string {
	return string(e)
}

func (e *Encoding) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Encoding(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Encoding", str)
	}
	return nil
}

func (e Encoding) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// specifies how a file is to be opened
type FileOpen string

const (
	// create file if it doesn't exist
	FileOpenCreate FileOpen = "create"
	// file must not exist
	FileOpenNew FileOpen = "new"
	// truncate the file if it exists
	FileOpenTruncate FileOpen = "truncate"
	// writes will be appended to the end of the file
	FileOpenAppend FileOpen = "append"
)

func (e FileOpen) IsValid() bool {
	switch e {
	case FileOpenCreate, FileOpenNew, FileOpenTruncate, FileOpenAppend:
		return true
	}
	return false
}

func (e FileOpen) String() string {
	return string(e)
}

func (e *FileOpen) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = FileOpen(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid FileOpen", str)
	}
	return nil
}

func (e FileOpen) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type FileType string

const (
	FileTypeRegular    FileType = "regular"
	FileTypeDir        FileType = "dir"
	FileTypeSymlink    FileType = "symlink"
	FileTypeNamedPipe  FileType = "namedPipe"
	FileTypeSocket     FileType = "socket"
	FileTypeDevice     FileType = "device"
	FileTypeCharDevice FileType = "charDevice"
	FileTypeIrregular  FileType = "irregular"
)

func (e FileType) IsValid() bool {
	switch e {
	case FileTypeRegular, FileTypeDir, FileTypeSymlink, FileTypeNamedPipe, FileTypeSocket, FileTypeDevice, FileTypeCharDevice, FileTypeIrregular:
		return true
	}
	return false
}

func (e FileType) String() string {
	return string(e)
}

func (e *FileType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = FileType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid FileType", str)
	}
	return nil
}

func (e FileType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
