package common

import "sync"
import "log"

var (
	hostMapping = make(map[string]string)
	hmLock      = &sync.RWMutex{}
)

func GetMappedHost(host string) string {
	hmLock.RLock()
	defer hmLock.RUnlock()
	if mapped, ok := hostMapping[host]; ok {
		log.Println("host map:", host, ">>>", mapped)
		return mapped
	}
	return host
}

func AddHostMapping(host, target string) {
	hmLock.Lock()
	defer hmLock.Unlock()
	hostMapping[host] = target
}

func DelHostMapping(host string) {
	hmLock.Lock()
	defer hmLock.Unlock()
	delete(hostMapping, host)
}

func ListHostMapping() (mapping map[string]string) {
	hmLock.RLock()
	defer hmLock.RUnlock()
	mapping = make(map[string]string)
	for k, v := range hostMapping {
		mapping[k] = v
	}
	return
}
