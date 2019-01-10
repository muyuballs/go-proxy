package common

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"runtime"
	"sync"
	"time"
)

type Flusher interface {
	Flush()
}

const (
	K = float64(1024)
	M = 1024 * K
	G = 1024 * M
)

func FormatNS(raw float64) string {
	if raw/G > 1 {
		return fmt.Sprintf("%.2fG", raw/G)
	}
	if raw/M > 1 {
		return fmt.Sprintf("%.2fM", raw/M)
	}
	if raw/K > 1 {
		return fmt.Sprintf("%.2fK", raw/K)
	}
	return fmt.Sprintf("%.2fB", raw)
}

func Transfer(destination io.WriteCloser, source io.ReadCloser, flow string) {
	if dacs, ok := destination.(*ACStream); ok {
		defer dacs.CloseW()
	} else {
		defer destination.Close()
	}
	if sacs, ok := source.(*ACStream); ok {
		defer sacs.CloseR()
	} else {
		defer source.Close()
	}
	startTime := time.Now()
	n, err := io.Copy(destination, source)
	cost := time.Since(startTime)
	log.Printf("%v %v %v %v/s %v --> %v\n", flow, n, FormatNS(float64(n)), FormatNS(float64(n)/cost.Seconds()), cost, err)
}

func CopyN(dst io.Writer, src io.Reader, size int) (n int, err error) {
	cpSize := 0
	buf := make([]byte, 8192)
	for size == -1 || size > 0 {
		if size != -1 && size < len(buf) {
			buf = make([]byte, size)
		}
		x, e := src.Read(buf)
		if e != nil {
			if e == io.EOF {
				break
			}
			return cpSize, e
		}
		cpSize += x
		_, e = dst.Write(buf[:x])
		if e != nil {
			return cpSize, e
		}
		size -= x
	}
	return cpSize, nil
}

func ReadByte(r io.Reader) (rel byte, err error) {
	buf := make([]byte, 1)
	_, err = io.ReadFull(r, buf)
	if err == nil {
		rel = buf[0]
	}
	return
}

func ReadFull(r io.Reader, buf []byte) error {
	var tn int = 0
	for tn < len(buf) {
		n, err := r.Read(buf[tn:])
		if err != nil {
			return err
		}
		tn += n
	}
	return nil
}

var (
	aliveAcs  = make(map[int]*ACStream)
	gAcsIndex = 0
	glock     = &sync.Mutex{}
)

type ACStream struct {
	Index  int
	origin interface{}
	r      *bufio.Reader
	w      *bufio.Writer
	c      io.Closer
	refs   uint32
	lock   *sync.Mutex
}

func callFunc(origin interface{}, name string, args ...interface{}) (rel []interface{}, succ bool) {
	vc := reflect.ValueOf(origin)
	crm := vc.MethodByName(name)
	if crm.IsValid() {
		params := make([]reflect.Value, 0)
		if args != nil {
			for _, x := range args {
				params = append(params, reflect.ValueOf(x))
			}
		}
		rels := crm.Call(params)
		rel := make([]interface{}, 0)
		for _, r := range rels {
			rel = append(rel, r.Interface())
		}
		return rel, true
	}
	return nil, false
}

//net.Conn
func (acs *ACStream) LocalAddr() net.Addr {
	rel, su := callFunc(acs.origin, "LocalAddr")
	if su && len(rel) > 0 && rel[0] != nil {
		return rel[0].(net.Addr)
	}
	return nil
}

func (acs *ACStream) RemoteAddr() net.Addr {
	rel, su := callFunc(acs.origin, "RemoteAddr")
	if su && len(rel) > 0 && rel[0] != nil {
		return rel[0].(net.Addr)
	}
	return nil
}

func (acs *ACStream) SetDeadline(t time.Time) error {
	rel, su := callFunc(acs.origin, "SetDeadline", t)
	if su && len(rel) > 0 && rel[0] != nil {
		return rel[0].(error)
	}
	return nil
}

func (acs *ACStream) SetReadDeadline(t time.Time) error {
	rel, su := callFunc(acs.origin, "SetReadDeadline", t)
	if su && len(rel) > 0 && rel[0] != nil {
		return rel[0].(error)
	}
	return nil
}

func (acs *ACStream) SetWriteDeadline(t time.Time) error {
	rel, su := callFunc(acs.origin, "SetWriteDeadline", t)
	if su && len(rel) > 0 && rel[0] != nil {
		return rel[0].(error)
	}
	return nil
}

func (acs *ACStream) Origin() interface{} {
	return acs.origin
}

func (acs *ACStream) Reader() *bufio.Reader {
	return acs.r
}

func (acs *ACStream) ReadLine() (line []byte, isPrefix bool, err error) {
	return acs.r.ReadLine()
}

func (acs *ACStream) Discard(n int) (discarded int, err error) {
	return acs.r.Discard(n)
}

func (acs *ACStream) Writer() *bufio.Writer {
	return acs.w
}

//

func (acs *ACStream) CloseR() error {
	callFunc(acs.origin, "CloseRead")
	return acs.Close()
}

func (acs *ACStream) CloseW() error {
	callFunc(acs.origin, "CloseWrite")
	return acs.Close()
}

func (acs *ACStream) Destroy() {
	acs.lock.Lock()
	defer acs.lock.Unlock()
	acs.refs = 0
	glock.Lock()
	defer glock.Unlock()
	delete(aliveAcs, acs.Index)
}

func (acs *ACStream) Close() error {
	acs.lock.Lock()
	defer acs.lock.Unlock()
	if acs.w != nil {
		acs.w.Flush()
	}
	acs.refs -= 1
	if acs.refs == 0 {
		glock.Lock()
		defer glock.Unlock()
		delete(aliveAcs, acs.Index)
		return acs.c.Close()
	}
	return nil
}

func (acs *ACStream) Open() *ACStream {
	acs.lock.Lock()
	defer acs.lock.Unlock()
	acs.refs += 1
	return acs
}

func (acs *ACStream) Flush() {
	if acs.w != nil {
		acs.w.Flush()
	}
}

func (acs *ACStream) Pick(n int) ([]byte, error) {
	return acs.r.Peek(n)
}

func (acs *ACStream) Read(buf []byte) (int, error) {
	if acs.r != nil {
		n, e := acs.r.Read(buf)
		return n, e
	}
	return 0, io.ErrUnexpectedEOF
}

func (acs *ACStream) Write(buf []byte) (int, error) {
	if acs.w != nil {
		return acs.w.Write(buf)
	}
	return 0, io.ErrShortWrite
}

func NewACS(base io.ReadWriteCloser) *ACStream {
	if acs, ok := base.(*ACStream); ok {
		return acs
	}
	glock.Lock()
	defer glock.Unlock()
	gAcsIndex += 1
	acs := &ACStream{
		Index:  gAcsIndex,
		origin: base,
		r:      bufio.NewReader(base),
		w:      bufio.NewWriterSize(base, 10),
		c:      base,
		refs:   1,
		lock:   &sync.Mutex{},
	}
	aliveAcs[gAcsIndex] = acs
	return acs
}

func PrintAcs() {
	glock.Lock()
	defer glock.Unlock()
	log.Printf("======[%d]ACS[%d]======\n", runtime.NumGoroutine(), len(aliveAcs))
	for k, v := range aliveAcs {
		log.Printf("%8d -> %d\n", k, v.refs)
	}
	log.Printf("======[%d]ACS[%d]======\n", runtime.NumGoroutine(), len(aliveAcs))
}
