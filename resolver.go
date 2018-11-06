//go:generate gqlgen

package fsgraph

import (
	context "context"
	"os"
	"path"

	"github.com/pkg/errors"
)

type Resolver struct {
	RootFS FS
}

func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}

func (r *Resolver) FileResult() FileResultResolver {
	return &fileResultResolver{r}
}

func (r *Resolver) RegularFile() RegularFileResolver {
	return &regularFileResolver{r}
}

func (r *Resolver) Dir() DirResolver {
	return &dirResolver{r}
}

func (r *Resolver) Internal_OtherFile() Internal_OtherFileResolver {
	return &internal_OtherFileResolver{r}
}

type fileResultResolver struct{ *Resolver }

func (r *fileResultResolver) File(ctx context.Context, obj *FileResult) (File, error) {
	if obj.path == "" {
		return nil, errors.New("not a file")
	}
	return r.RootFS.GetFile(obj.path)
}

type regularFileResolver struct{ *Resolver }

func (r *regularFileResolver) Parent(ctx context.Context, obj *RegularFile) (File, error) {
	return obj.getParent()
}

func (r *regularFileResolver) Contents(ctx context.Context, obj *RegularFile, encoding Encoding, maxReadBytes Int64, seek Int64) (FileContents, error) {
	return obj.getContents(ctx, encoding, int64(maxReadBytes), int64(seek))
}

type dirResolver struct{ *Resolver }

func (r *dirResolver) Parent(ctx context.Context, obj *Dir) (File, error) {
	return obj.getParent()
}

func (r *dirResolver) Children(ctx context.Context, obj *Dir, first int) ([]File, error) {
	return obj.getChildren(ctx, first)
}

func (r *dirResolver) File(ctx context.Context, obj *Dir, apath string) (File, error) {
	f, err := r.RootFS.GetFile(path.Join(obj.Path, path.Clean(apath)))
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return f, err
}

type internal_OtherFileResolver struct{ *Resolver }

func (r *internal_OtherFileResolver) Parent(ctx context.Context, obj *Internal_OtherFile) (File, error) {
	return obj.getParent()
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) Remove(ctx context.Context, path string) (OKResult, error) {
	err := r.RootFS.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			warning := err.Error()
			return OKResult{Warning: &warning}, nil
		}
		return OKResult{}, err
	}
	return OKResult{S: "removed"}, nil
}
func (r *mutationResolver) Rename(ctx context.Context, apath string, anewName string) (FileResult, error) {
	err := r.RootFS.Rename(apath, anewName)
	if err != nil {
		return FileResult{}, err
	}
	newpath := path.Join(path.Dir(apath), anewName)
	return FileResult{S: "renamed", path: newpath}, nil
}
func (r *mutationResolver) Chmod(ctx context.Context, path string, mode int) (FileResult, error) {
	err := r.RootFS.Chmod(path, os.FileMode(mode)&os.ModePerm)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{S: "mode changed", path: path}, nil
}
func (r *mutationResolver) Write(ctx context.Context, path string, contents string, open []FileOpen, encoding Encoding) (FileResult, error) {
	openflags, err := fileOpenFlags(open)
	if err != nil {
		return FileResult{}, err
	}
	openflags |= os.O_WRONLY
	f, err := r.RootFS.OpenFile(path, openflags, 0666)
	if err != nil {
		return FileResult{}, err
	}
	defer f.Close()
	err = fileWrite(f, contents, encoding)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{S: "file written", path: path}, nil
}
func (r *mutationResolver) Mkdir(ctx context.Context, path string) (FileResult, error) {
	err := r.RootFS.Mkdir(path, 0777)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{S: "directory created", path: path}, nil
}
func (r *mutationResolver) MkdirAll(ctx context.Context, path string) (FileResult, error) {
	err := r.RootFS.MkdirAll(path, 0777)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{S: "directory created", path: path}, nil
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Root(ctx context.Context) (Dir, error) {
	return r.RootFS.GetDir("/")
}
func (r *queryResolver) Cd(ctx context.Context, path string) (*Dir, error) {
	d, err := r.RootFS.GetDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}
func (r *queryResolver) File(ctx context.Context, path string) (File, error) {
	f, err := r.RootFS.GetFile(path)
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return f, err
}
