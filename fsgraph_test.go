package fsgraph

import (
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/99designs/gqlgen/handler"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestFSGraph(t *testing.T) {
	//rootfs := afero.NewMemMapFs()
	tempdir, err := ioutil.TempDir("", "fsgraph-test")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tempdir)
	rootfs := afero.NewBasePathFs(afero.NewOsFs(), tempdir)

	// setup some files, dirs and data:
	file1path := "/file1"
	file1content := []byte(`File one.`)
	afero.WriteFile(rootfs, file1path, file1content, 0666)
	file2path := "/file2"
	file2content := []byte(`File two.`)
	afero.WriteFile(rootfs, file2path, file2content, 0666)
	rootfs.MkdirAll("a/b/c", 0777)
	file3path := "/a/b/c/file3"
	file3content := []byte(`File three.`)
	afero.WriteFile(rootfs, file3path, file3content, 0666)

	srv := httptest.NewServer(handler.GraphQL(NewExecutableSchema(Config{
		Resolvers: &Resolver{
			RootFS: FS{Fs: rootfs},
		},
	})))
	c := client.New(srv.URL)

	var resp struct {
		Root struct {
			Path     string `json:"path"`
			Children []struct {
				Path string `json:"path"`
				Mode struct {
					Type string `json:"type"`
				} `json:"mode"`
			} `json:"children"`
		} `json:"root"`
	}
	c.MustPost(`query { root { path, children { path, mode{type} } } }`, &resp)
	//t.Log(spew.Sdump(resp))
	require.Equal(t, "/", resp.Root.Path, "root path")
	// NOTE: afero.NewMemMapFs() is showing "/a" twice, both as a regular file...
	require.Equal(t, 3, len(resp.Root.Children), "length of root's children")

}
