package http

import (
	"log"
	"regexp"
	"strings"
	"sync"
)

type Rdt int8
type Fbt int8

const (
	RedirectFile = Rdt(iota)
	RedirectFolder
	RedirectCustom

	FallbackToSource = Fbt(iota)
	FallbackTo404
)

type RedirectItem struct {
	Type        Rdt
	Url         string
	Target      string
	Headers     map[string]string
	Body        string
	ContentType string
	Fallback    Fbt
}

var (
	customMapping = make(map[*regexp.Regexp]*RedirectItem)
	fileMapping   = make(map[*regexp.Regexp]*RedirectItem)
	folderMapping = make(map[string]*RedirectItem)
	pmLocl        = &sync.RWMutex{}
)

func GetMappedPath(path string) *RedirectItem {
	pmLocl.RLock()
	defer pmLocl.RUnlock()
	for r, v := range customMapping {
		if r.MatchString(path) {
			log.Println("custom map:", path, ">>>", v)
			return v
		}
	}
	for r, v := range fileMapping {
		if r.MatchString(path) {
			log.Println("file map:", path, ">>>", v)
			return v
		}
	}

	for r, v := range folderMapping {
		if strings.HasPrefix(path, r) {
			log.Println("file map:", path, ">>>", v)
			return v
		}
	}
	return nil
}

func AddCustomMapping(expr, code, body, contentType string, headers map[string]string) (err error) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	pr, err := regexp.Compile(expr)
	if err != nil {
		return
	}
	customMapping[pr] = &RedirectItem{
		Type:        RedirectCustom,
		Url:         expr,
		Target:      code,
		Headers:     headers,
		Body:        body,
		ContentType: contentType,
	}
	return
}

func DelCustomMapping(expr string) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	for r, i := range customMapping {
		if i.Url == expr {
			delete(customMapping, r)
			break
		}
	}
}

func AddFileMapping(expr, target string, fbt Fbt) (err error) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	pr, err := regexp.Compile(expr)
	if err != nil {
		return
	}
	fileMapping[pr] = &RedirectItem{
		Type:     RedirectFile,
		Url:      expr,
		Target:   target,
		Fallback: fbt,
	}
	return
}

func DelFileMapping(expr string) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	for r, i := range fileMapping {
		if i.Url == expr {
			delete(fileMapping, r)
			break
		}
	}
}

func AddFolderMapping(expr, folder string, fbt Fbt) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	folderMapping[expr] = &RedirectItem{
		Type:     RedirectFolder,
		Url:      expr,
		Target:   folder,
		Fallback: fbt,
	}
}

func DelFolderMapping(expr string) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	delete(folderMapping, expr)
}
