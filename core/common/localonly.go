package common

import (
	"bufio"
	"github.com/fsnotify/fsnotify"
	"io"
	"log"
	"os"
	"regexp"
	"sync"
)

var (
	localOnlyList = make([]*regexp.Regexp, 0)
	lolLock       = &sync.RWMutex{}
	parseLock     = &sync.Mutex{}
)

func IsLocalOnly(host string) bool {
	lolLock.RLock()
	defer lolLock.RUnlock()
	if len(localOnlyList) == 0 {
		return false
	}
	for _, r := range localOnlyList {
		if r.MatchString(host) {
			return true
		}
	}
	return false
}

func AddLocalOnly(expr string) (err error) {
	lolLock.Lock()
	defer lolLock.Unlock()
	r, err := regexp.Compile(string(expr))
	if err != nil {
		return
	}
	localOnlyList = append(localOnlyList, r)
	return
}

func DelLocalOnly(expr *regexp.Regexp) {
	lolLock.Lock()
	defer lolLock.Unlock()
	for i := range localOnlyList {
		if localOnlyList[i] == expr {
			localOnlyList = append(localOnlyList[:i], localOnlyList[i+1:]...)
			break
		}
	}
}

func ListLocalOnly() (list []*regexp.Regexp) {
	lolLock.RLock()
	defer lolLock.RUnlock()
	list = make([]*regexp.Regexp, 0)
	for _, r := range localOnlyList {
		list = append(list, r)
	}
	return
}

func ParseLolFile(lolFile string) {
	defer func() {
		for _, r := range localOnlyList {
			log.Println(r.String())
		}
	}()
	parseLock.Lock()
	defer parseLock.Unlock()
	log.Println("parse localOnlyList file")
	f, err := os.Open(lolFile)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()
	tlol := make([]*regexp.Regexp, 0)
	reader := bufio.NewReader(f)
	for {
		ld, _, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Println(err)
				break
			}
		}
		r, err := regexp.Compile(string(ld))
		if err != nil {
			log.Println(err)
			continue
		}
		tlol = append(tlol, r)
	}
	lolLock.Lock()
	defer lolLock.Unlock()
	localOnlyList = tlol
}

func LoadLol(conf *Config) error {
	if len(conf.LolFile) > 0 {
		log.Println("load localOnlyList list")
		defer log.Println("localOnlyList list load done.")
		ParseLolFile(conf.LolFile)
		go func() {
			w, err := fsnotify.NewWatcher()
			if err != nil {
				log.Println(err)
				return
			}
			defer w.Close()
			err = w.Add(conf.LolFile)
			if err != nil {
				log.Println(err)
				return
			}
			for {
				select {
				case event := <-w.Events:
					if event.Op&fsnotify.Write == fsnotify.Write {
						ParseLolFile(conf.LolFile)
					}
				case err := <-w.Errors:
					log.Println(err)
					return
				}
			}
		}()
	}
	return nil
}
