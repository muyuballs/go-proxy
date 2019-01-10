package http

import (
	"encoding/json"
	"github.com/muyuballs/go-proxy/core/common"
	"hash/crc32"
	"log"
	"strconv"
	"time"
)

type FileInfo struct {
	Key         string
	Name        string
	Size        int
	ContentType string
}

type RequestInfo struct {
	Version     string
	Protocol    string
	Method      string
	Host        string
	FullUrl     string
	Url         string
	Headers     map[string]string
	Query       map[string]string
	WebForm     map[string]string
	Files       []*FileInfo
	ContentType string
	Body        []byte
}

type ResponseInfo struct {
	Status      int
	Version     string
	Message     string
	BodySize    int
	ContextType string
	Headers     map[string]string
}

type SessionInfo struct {
	Sid          string
	BeginTime    time.Time
	EndTime      time.Time
	RemoteAddr   string
	RemotePort   int
	RequestInfo  *RequestInfo
	ResponseInfo *ResponseInfo
	Done         bool
	endChan      chan int
}

func NewSessionInfo(conf *common.Config) *SessionInfo {
	sif := &SessionInfo{
		Sid:       strconv.FormatInt(time.Now().UnixNano(), 16),
		BeginTime: time.Now(),
		endChan:   make(chan int),
	}
	if conf.LogChan != nil {
		go func() {
			var oldHash uint32 = 0
			for {
				select {
				case <-conf.Context.Done():
					return
				case _, _ = <-sif.endChan:
					sendLogToChan(conf, sif)
					log.Println("session done")
					return
				case <-time.Tick(time.Second):
					nHash := crc32WithGob(sif)
					if nHash != oldHash {
						oldHash = nHash
						sendLogToChan(conf, sif)
					}
				}
			}
		}()
	}
	return sif
}

func (s *SessionInfo) SessionDone() {
	s.EndTime = time.Now()
	s.Done = true
	close(s.endChan)
}

func crc32WithGob(v interface{}) uint32 {
	data, _ := json.Marshal(v)
	return crc32.ChecksumIEEE(data)
}

func sendLogToChan(conf *common.Config, sif *SessionInfo) {
	if conf.LogChan == nil || sif == nil {
		return
	}
	go func() {
		select {
		case <-conf.Context.Done():
			break
		case conf.LogChan <- sif:
			return
		case <-time.Tick(time.Millisecond * 10):
			log.Println("send failed")
			return
		}
	}()
}
