package http

import (
	"log"
	"regexp"
	"sync"
)

var (
	pathMapping = make(map[*regexp.Regexp]string)
	pmLocl      = &sync.RWMutex{}
)

func GetMappedPath(path string) string {
	pmLocl.RLock()
	defer pmLocl.RUnlock()
	for r, v := range pathMapping {
		if r.MatchString(path) {
			log.Println("path map:", path, ">>>", v)
			return v
		}
	}
	return path
}

func AddPathMapping(expr, target string) (err error) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	pr, err := regexp.Compile(expr)
	if err != nil {
		return
	}
	pathMapping[pr] = target
	return
}

func DelPathMapping(r *regexp.Regexp) {
	pmLocl.Lock()
	defer pmLocl.Unlock()
	delete(pathMapping, r)
}

func ListPathMapping() (mapping map[*regexp.Regexp]string) {
	pmLocl.RLock()
	defer pmLocl.RUnlock()
	mapping = make(map[*regexp.Regexp]string)
	for k, v := range pathMapping {
		mapping[k] = v
	}
	return
}
