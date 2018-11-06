package fsgraph

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type FS struct {
	afero.Fs
	Scope []byte
}

func (fs FS) genID(path string) string {
	sb := &strings.Builder{}
	enc := base64.NewEncoder(base64.StdEncoding, sb)
	enc.Write(fs.Scope)
	io.WriteString(enc, path)
	enc.Close()
	return sb.String()
}

func (fs FS) ID() string {
	return fs.genID("")
}

func (fs FS) GetFile(path string) (File, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return nil, err
	}
	return getFsFileFromInfo(path, fi, fs)
}

func (fs FS) GetDir(dirPath string) (Dir, error) {
	fi, err := fs.Stat(dirPath)
	if err != nil {
		return Dir{}, err
	}
	if !fi.IsDir() {
		return Dir{}, errors.New("Not a directory")
	}
	return Dir{makeFileBase(dirPath, fi, fs)}, nil
}

type fileBase struct {
	Path string `json:"path"`
	os.FileInfo
	fs FS
}

func (fileBase) IsFile() {}

func (fb fileBase) ID() string {
	return fb.fs.genID(fb.Path)
}

func (fb fileBase) Name() string {
	if fb.Path == "/" {
		return ""
	}
	name := fb.FileInfo.Name()
	if name == "." {
		name = ""
	}
	return name
}

func (fb fileBase) Size() Int64 {
	return Int64(fb.FileInfo.Size())
}

func fileTypeFromOsFileMode(fm os.FileMode) FileType {
	switch fm & (os.ModeType | os.ModeCharDevice) {
	case 0:
		return FileTypeRegular
	case os.ModeDir:
		return FileTypeDir
	case os.ModeSymlink:
		return FileTypeSymlink
	case os.ModeNamedPipe:
		return FileTypeNamedPipe
	case os.ModeSocket:
		return FileTypeSocket
	case os.ModeDevice:
		return FileTypeDevice
	case os.ModeDevice | os.ModeCharDevice:
		return FileTypeCharDevice
	default:
		return FileTypeIrregular
	}
}

func fileModeFromOsFileInfo(fi os.FileInfo) FileMode {
	return FileMode{
		Type:   fileTypeFromOsFileMode(fi.Mode()),
		Perm:   int(fi.Mode() & os.ModePerm),
		Sticky: (fi.Mode() & os.ModeSticky) == os.ModeSticky,
	}
}

func (fb fileBase) Mode() FileMode {
	return fileModeFromOsFileInfo(fb.FileInfo)
}

const timeFmt = "2006-01-02T15:04:05.000Z07:00"

func (fb fileBase) ModTime() string {
	return fb.FileInfo.ModTime().UTC().Format(timeFmt)
}

func (fb fileBase) getParent() (File, error) {
	ppath := path.Join(fb.Path, "..")
	if ppath == fb.Path {
		return nil, nil // no parent
	}
	fi, err := fb.fs.Stat(ppath)
	if err != nil {
		return nil, err
	}
	return Dir{makeFileBase(ppath, fi, fb.fs)}, nil
}

func makeFileBase(path string, fi os.FileInfo, fs FS) fileBase {
	if path == "." || path == "" {
		path = "/"
	} else if path[0] != '/' {
		path = "/" + path
	}
	return fileBase{path, fi, fs}
}

func getFsFileFromInfo(path string, fi os.FileInfo, fs FS) (File, error) {
	switch fi.Mode() & (os.ModeType | os.ModeCharDevice) {
	case 0: // regular:
		return RegularFile{makeFileBase(path, fi, fs)}, nil
	case os.ModeDir:
		return Dir{makeFileBase(path, fi, fs)}, nil
	default:
		return Internal_OtherFile{makeFileBase(path, fi, fs)}, nil
	}
}

// ErrInvalidEncoding is returned if the Encoding is not valid.
var ErrInvalidEncoding = errors.New("Invalid encoding")

type RegularFile struct {
	fileBase
}

func (rf RegularFile) getContents(ctx context.Context, encoding Encoding, maxReadBytes int64, seek int64) (FileContents, error) {
	f, err := rf.fs.Open(rf.Path)
	if err != nil {
		return FileContents{}, err
	}
	defer f.Close()

	if seek >= 0 {
		_, err := f.Seek(seek, os.SEEK_SET)
		if err != nil {
			return FileContents{}, err
		}
	}

	maxReadAllowed := int64(1024 * 1024 * 8)
	srclimit := maxReadAllowed
	if maxReadBytes >= 0 && int64(maxReadBytes) < srclimit {
		srclimit = maxReadBytes
	}
	src := io.LimitReader(f, srclimit)

	var fc FileContents
	fc.Encoding = encoding
	var nreadbytes int64
	switch encoding {
	case EncodingUtf8, EncodingAuto: // If auto, prefer utf8 as it's less work and less data.
		buf := &strings.Builder{}
		nreadbytes, err = io.Copy(buf, src)
		if err != nil {
			return FileContents{}, err
		}
		s := buf.String()
		// TODO: consider utf8 "BOM"?
		if utf8.ValidString(s) {
			fc.Data = s
			fc.Encoding = EncodingUtf8
			if encoding == EncodingAuto {
				// Auto encoding: it is valid utf8, but see if it looks like binary...
				// This is somewhat magic and not guaranteed to return a consistent encoding,
				// but if you handle both utf8 and base64 you'll get the correct content (if no warnings)
				// Note: If you always want a specific encoding or consistency, don't use auto.
				nlow := 0
				nnul := 0
				for i := 0; i < len(s); i++ {
					b := s[i]
					if b < 32 && b != '\n' && b != '\r' && b != '\t' && b != '\v' {
						nlow++
						if b == 0 {
							nnul++
						}
					}
				}
				// Decide if it looks like binary. This could probably use some tweaking.
				if nlow > 0 && (nnul > len(s)/16 || nlow >= len(s)/4) {
					fc.Data = base64.StdEncoding.EncodeToString([]byte(s))
					fc.Encoding = EncodingBase64
				}
			}
		} else {
			if encoding == EncodingAuto { // Not utf8, auto switch to base64.
				fc.Data = base64.StdEncoding.EncodeToString([]byte(s))
				fc.Encoding = EncodingBase64
			} else { // Invalid utf8 with EncodingUtf8...
				fc.Data = strings.Map(func(r rune) rune { // "Clean up" (modify) the string.
					if r == utf8.RuneError {
						return unicode.ReplacementChar
					}
					return r
				}, s)
				warning := "Invalid UTF-8 encountered"
				fc.Warning = &warning
			}
		}
	case EncodingBase64:
		buf := &bytes.Buffer{}
		nreadbytes, err = io.Copy(buf, src)
		if err != nil {
			return FileContents{}, err
		}
		fc.Data = base64.StdEncoding.EncodeToString(buf.Bytes())
	default:
		return FileContents{}, ErrInvalidEncoding
	}

	moredata := nreadbytes == srclimit
	if moredata {
		// If we read the limit, see if the file is bigger.
		// TODO: check for EOF on f, as Stat may be outdated and slower.
		st, err := f.Stat()
		if err != nil {
			// TODO: Consider attempting to read another byte from f.
			warning := "Unable to determine file size"
			fc.Warning = &warning
		} else {
			if st.Size() > nreadbytes {
				/*if nreadbytes == maxReadAllowed && maxReadBytes < srclimit {
					warning := "Content too large"
					fc.Warning = &warning
				}*/
			} else {
				moredata = false
			}
		}
	}

	if moredata {
		pos, err := f.Seek(0, os.SEEK_CUR)
		if err == nil {
			fcnext := Int64(pos)
			fc.Next = &fcnext
		}
	}

	return fc, nil
}

type Dir struct {
	fileBase
}

func (dir Dir) getChildren(ctx context.Context, first int) ([]File, error) {
	f, err := dir.fs.Open(dir.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	list, err := f.Readdir(-1)
	if err != nil {
		return nil, err
	}
	if first >= 0 && len(list) > first {
		list = list[:first]
	}
	var results []File
	for _, fi := range list {
		path := path.Join(dir.Path, fi.Name())
		fx, err := getFsFileFromInfo(path, fi, dir.fs)
		if err != nil {
			return nil, err
		}
		results = append(results, fx)
	}
	return results, nil
}

type Internal_OtherFile struct {
	fileBase
}

type FileResult struct {
	S       string  `json:"s"`
	Warning *string `json:"warning"`
	path    string
}

func (FileResult) IsResult() {}

// Returns the flags for OpenFile, or an error.
// One of the read/write flags will also need to be combined with the result.
func fileOpenFlags(open []FileOpen) (int, error) {
	flags := 0
	for _, op := range open {
		switch op {
		case FileOpenCreate:
			flags |= os.O_CREATE
		case FileOpenNew:
			flags |= os.O_EXCL | os.O_CREATE
		case FileOpenTruncate:
			flags |= os.O_TRUNC
		case FileOpenAppend:
			flags |= os.O_APPEND
		default:
			return 0, errors.New("Invalid FileOpen value: " + string(op))
		}
	}
	return flags, nil
}

func fileWrite(f afero.File, contents string, encoding Encoding) error {
	switch encoding {
	case EncodingUtf8:
		// We'll assume it's valid utf8.
		_, err := f.WriteString(contents)
		return err
	case EncodingBase64:
		data, err := base64.StdEncoding.DecodeString(contents)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	default:
		return ErrInvalidEncoding
	}
}
