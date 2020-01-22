package webdav

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/emersion/go-webdav/internal"
)

type LocalFileSystem string

func (fs LocalFileSystem) path(name string) (string, error) {
	if (filepath.Separator != '/' && strings.IndexRune(name, filepath.Separator) >= 0) || strings.Contains(name, "\x00") {
		return "", internal.HTTPErrorf(http.StatusBadRequest, "webdav: invalid character in path")
	}
	name = path.Clean(name)
	if !path.IsAbs(name) {
		return "", internal.HTTPErrorf(http.StatusBadRequest, "webdav: expected absolute path")
	}
	return filepath.Join(string(fs), filepath.FromSlash(name)), nil
}

func (fs LocalFileSystem) href(path string) (string, error) {
	rel, err := filepath.Rel(string(fs), path)
	if err != nil {
		return "", err
	}
	href := "/" + filepath.ToSlash(rel)
	return href, nil
}

func (fs LocalFileSystem) Open(name string) (io.ReadCloser, error) {
	p, err := fs.path(name)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func fileInfoFromOS(href string, fi os.FileInfo) *FileInfo {
	return &FileInfo{
		Href:    href,
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		IsDir:   fi.IsDir(),
		// TODO: fallback to http.DetectContentType?
		MIMEType: mime.TypeByExtension(path.Ext(href)),
	}
}

func (fs LocalFileSystem) Stat(name string) (*FileInfo, error) {
	p, err := fs.path(name)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	return fileInfoFromOS(name, fi), nil
}

func (fs LocalFileSystem) Readdir(name string, recursive bool) ([]FileInfo, error) {
	p, err := fs.path(name)
	if err != nil {
		return nil, err
	}

	var l []FileInfo
	err = filepath.Walk(p, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		href, err := fs.href(p)
		if err != nil {
			return err
		}

		l = append(l, *fileInfoFromOS(href, fi))

		if !recursive && fi.IsDir() {
			return filepath.SkipDir
		}
		return nil
	})
	return l, err
}

func (fs LocalFileSystem) Create(name string) (io.WriteCloser, error) {
	p, err := fs.path(name)
	if err != nil {
		return nil, err
	}
	return os.Create(p)
}

func (fs LocalFileSystem) RemoveAll(name string) error {
	p, err := fs.path(name)
	if err != nil {
		return err
	}

	// WebDAV semantics are that it should return a "404 Not Found" error in
	// case the resource doesn't exist. We need to Stat before RemoveAll.
	if _, err = os.Stat(p); err != nil {
		return err
	}

	return os.RemoveAll(p)
}

func (fs LocalFileSystem) Mkdir(name string) error {
	p, err := fs.path(name)
	if err != nil {
		return err
	}
	return os.Mkdir(p, 0755)
}

var _ FileSystem = LocalFileSystem("")
