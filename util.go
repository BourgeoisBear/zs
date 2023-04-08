package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func errRpt(err error, isTty bool) {
	if err != nil {
		if isTty {
			fmt.Fprint(os.Stderr, "\x1b[91;1mERROR\x1b[0m: ")
		} else {
			fmt.Fprint(os.Stderr, "ERROR: ")
		}
		fmt.Fprintln(os.Stderr, err.Error())
	}
}

func searchDirAncestors(start, needle string) (found string, err error) {

	start, err = filepath.Abs(start)
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			err = errors.WithMessagef(err, "search for `%s` in `%s` ancestors", needle, start)
		}
	}()

	cur := start
	inf, err := os.Stat(cur)
	if err != nil {
		return
	}

	if !inf.IsDir() {
		cur = filepath.Dir(cur)
	}

	if filepath.Base(cur) == needle {
		found = cur
		return
	}

	for {

		// check siblings
		var sD []os.DirEntry
		if sD, err = os.ReadDir(cur); err != nil {
			return
		}
		for ix := range sD {
			if sD[ix].IsDir() {
				sname := sD[ix].Name()
				if sname == needle {
					found = filepath.Join(cur, sname)
					return
				}
			}
		}

		// up a dir
		cur = filepath.Dir(cur)

		// exit at root
		if strings.HasSuffix(cur, string(filepath.Separator)) {
			break
		}
	}

	err = os.ErrNotExist
	return
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}

func HeadHandler(hDir http.Dir, oHandler http.Handler) http.Handler {

	return http.HandlerFunc(func(iWri http.ResponseWriter, pRq *http.Request) {

		if pRq.Method != "HEAD" {
			oHandler.ServeHTTP(iWri, pRq)
			return
		}

		var err error

		defer func() {
			if err == nil {
				iWri.WriteHeader(http.StatusOK)
				return
			}
			iWri.WriteHeader(http.StatusInternalServerError)
			iWri.Write([]byte(err.Error()))
		}()

		// PROCESS URI
		szPath := path.Clean(pRq.URL.Path)
		if szPath == "/" {
			szPath = "/index.html"
		}

		// OPEN FILE
		oFile, err := hDir.Open(szPath)
		if err != nil {
			return
		}
		defer oFile.Close()

		// GET THE CONTENT-TYPE OF THE FILE
		FileHeader := make([]byte, 512)
		oFile.Read(FileHeader)
		FileContentType := http.DetectContentType(FileHeader)

		// GET FILE SIZE
		FileStat, err := oFile.Stat()
		if err != nil {
			return
		}

		// WRITE HEADER
		iWri.Header().Set("Content-Type", FileContentType)
		iWri.Header().Set("Content-Length", strconv.FormatInt(FileStat.Size(), 10))
		iWri.Header().Set("Last-Modified", FileStat.ModTime().Format(time.RFC1123))
	})
}
