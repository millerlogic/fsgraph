# fsgraph
fsgraph is a GraphQL interface for a file system, built with [gqlgen](https://github.com/99designs/gqlgen).

Build it:

```
$ go get -v github.com/millerlogic/fsgraph/...
$ go install github.com/millerlogic/fsgraph/cmd/fsgraph
```

Usage:

```
$ fsgraph --help
Usage of fsgraph:
  -address string
    	HTTP address for the GraphQL server (default "localhost:8080")
  -protected
    	Writes go to a temporary location (default true)
  -root string
    	Root path of the file system to serve (default "/current/dir")
  -scope string
    	Set the file ID scope, before hashing (defaults to hostname:root)
```

Run:

```
$ fsgraph &
2018/11/06 20:30:22 FS root: /current/dir
2018/11/06 20:30:22 file ID scope: e3248d8392f13fa55fc3dc192ed4e793 (hashed from myhost:/current/dir)
2018/11/06 20:30:22 protected: temporary overlay dir: /tmp/fsgraph923227248
2018/11/06 20:30:22 connect to http://localhost:8080/ for GraphQL playground
```

By default it serves files from your current directory on localhost:8080 (only localhost can connect), and protected is enabled which means any writes will go to a separate temporary location.
Scope is used to create file IDs, as a way of attempting to make them global IDs. By default it is your computer's host name, a colon, and the root dir path, which all gets hashed.
All of these defaults can be overridden on the command line.
