package openapi

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed *.yaml components/* paths/*
var OpenAPIFS embed.FS

func FromFS(ctx context.Context, fsys fs.FS, path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		return fs.ReadFile(fsys, url.Path)
	}

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	if err := doc.Validate(ctx); err != nil {
		return nil, err
	}

	return doc, nil
}

func OpenAPIHandler() http.Handler {
	return http.FileServer(http.FS(OpenAPIFS))
}
