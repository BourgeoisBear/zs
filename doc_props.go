package main

import (
	tHtml "html/template"
	"io"
	"os"
	"regexp"
)

type DocProps struct {
	Path   string
	Info   os.FileInfo
	Body   []byte
	Vars   Vars
	Layout *tHtml.Template
}

/*
applies layout template in doc.Layout to bsContent, renders to iWri.

if doc.Layout == nil, bsContent is written directly to iWri.
*/
func (doc *DocProps) ApplyLayout(bsContent []byte, iWri io.Writer) error {

	if doc.Layout == nil {
		_, err := iWri.Write(bsContent)
		return err
	}

	/*
		TODO: backport page/layout changes to `default_conf`
		TODO: direct recursive copy of `default_conf`

		TODO: test syntax breakages in "real life"

		TODO: html/template: JS, CSS, et. al.

		TODO: l/r delim var options
			1. always ClearDelims, document ldelim/rdelim as unavailable in template
			2. keep layout delims separately, restore on layout render
			3. special varkey prefix for 'this file only' application
					CAPS KEYS as system-provided
					tolower everything in header parse
					what for header-parseable, this-file-only vars?
	*/

	// render
	vi := make(map[string]interface{}, len(doc.Vars)+1)
	for k, v := range doc.Vars {
		vi[k] = v
	}
	vi["HTML_CONTENT"] = tHtml.HTML(bsContent)
	// NOTE: re-populate Funcs() to bind updated Vars
	return doc.Layout.Funcs(funcMap(doc.Vars)).Execute(iWri, vi)
}

/*
retrieves file contents.  splits at first headerDelim.  parses text above headerDelim into DocProps.Vars.  returns text below headerDelim as DocProps.Body.

if no headerDelim is found, full contents are returned in DocProps.Body.
*/
func GetDoc(path, headerDelim string) (DocProps, error) {

	ret := DocProps{Path: path, Vars: make(Vars)}
	pf, err := os.Open(path)
	if err != nil {
		return ret, err
	}
	defer pf.Close()

	ret.Info, err = pf.Stat()
	if err != nil {
		return ret, err
	}

	ret.Body, err = io.ReadAll(pf)
	if err != nil {
		return ret, err
	}

	// find `headerDelim`
	// NOTE: deferring CR/CR-LF/first-line/last-line handling to regexp
	hdrPat := `(?:^|\r?\n)` + regexp.QuoteMeta(headerDelim) + `(?:$|\r?\n)`
	rx, err := regexp.Compile(hdrPat)
	if err != nil {
		return ret, err
	}
	hdrPos := rx.FindIndex(ret.Body)

	// not found, leave body as-is
	if hdrPos == nil {
		return ret, err
	}

	// found, parse vars from header info
	ret.Vars = ParseHeaderVars(ret.Body[:hdrPos[0]])
	ret.Body = ret.Body[hdrPos[1]:]
	return ret, nil
}