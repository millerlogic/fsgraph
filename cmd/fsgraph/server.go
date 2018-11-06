package main

import (
	"context"
	"crypto/sha512"
	"flag"
	"fmt"
	"io/ioutil"
	log "log"
	http "net/http"
	os "os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	fsgraph "github.com/millerlogic/fsgraph"

	"github.com/99designs/gqlgen/graphql"
	handler "github.com/99designs/gqlgen/handler"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/vektah/gqlparser/gqlerror"
)

func run() error {
	address := "localhost:8080"
	flag.StringVar(&address, "address", address, "HTTP address for the GraphQL server")
	fsrootdir, _ := os.Getwd()
	flag.StringVar(&fsrootdir, "root", fsrootdir, "Root path of the file system to serve")
	protected := true
	flag.BoolVar(&protected, "protected", protected, "Writes go to a temporary location")
	scopestr := ""
	flag.StringVar(&scopestr, "scope", scopestr, "Set the file ID scope, before hashing (defaults to hostname:root)")
	flag.Parse()
	if address == "" {
		flag.Usage()
		return errors.New("address expected")
	}
	if fsrootdir == "" {
		flag.Usage()
		return errors.New("root expected")
	}

	rootfs := afero.NewBasePathFs(afero.NewOsFs(), fsrootdir)
	log.Printf("FS root: %s", fsrootdir)

	if scopestr == "" {
		hostname, _ := os.Hostname()
		scopestr = hostname + ":" + fsrootdir
	}
	var scope []byte
	{
		xscope := sha512.Sum512([]byte(scopestr))
		scope = xscope[:16]
	}
	log.Printf("file ID scope: %x (hashed from %s)", scope, scopestr)

	if protected {
		// Wrap rootfs in a copy-on-write FS:
		tempdir, err := ioutil.TempDir("", "fsgraph")
		if err != nil {
			return err
		}
		defer func() {
			err := os.RemoveAll(tempdir)
			if err != nil {
				log.Printf("unable to clean up %v: %s", tempdir, err)
			} else {
				log.Printf("cleaned up %v", tempdir)
			}
		}()
		log.Printf("protected: temporary overlay dir: %v", tempdir)
		rootfs = afero.NewCopyOnWriteFs(rootfs, afero.NewBasePathFs(afero.NewOsFs(), tempdir))
	}

	http.Handle("/", handler.Playground("GraphQL playground", "/query"))
	http.Handle("/query",
		handler.GraphQL(
			fsgraph.NewExecutableSchema(fsgraph.Config{
				Resolvers: &fsgraph.Resolver{
					RootFS: fsgraph.FS{Fs: rootfs, Scope: scope},
				},
			}),
			handler.ErrorPresenter(func(ctx context.Context, err error) *gqlerror.Error {
				gqlerr := graphql.DefaultErrorPresenter(ctx, err)
				exts := make(map[string]interface{})
				if gqlerr.Extensions != nil {
					for k, v := range gqlerr.Extensions {
						exts[k] = v
					}
				}
				exts["gotype"] = strings.TrimLeft(fmt.Sprintf("%T", errors.Cause(err)), "*")
				gqlerr.Extensions = exts
				return gqlerr
			}),
		),
	)

	server := &http.Server{Addr: address, Handler: nil}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sigch := range sigchan {
			log.Printf("Received %s signal", sigch)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			server.Shutdown(ctx)
			break
		}
	}()

	connaddr := address
	if connaddr[0] == ':' {
		connaddr = "localhost" + connaddr
	}
	log.Printf("connect to http://%s/ for GraphQL playground", connaddr)
	serverErr := server.ListenAndServe()
	if serverErr == http.ErrServerClosed {
		log.Printf("%s", serverErr)
		serverErr = nil
	}

	close(sigchan)
	return serverErr
}

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}
