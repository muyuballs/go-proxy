package common

import (
	"io"
	"os"
)

type JournalReadWriter struct {
	io.ReadWriteCloser
	origin       io.ReadWriteCloser
	readJournal  *os.File
	writeJournal *os.File
}

func NewJRW(origin io.ReadWriteCloser, readJournal, writerJournal string) (jrw *JournalReadWriter, err error) {
	jrw = &JournalReadWriter{
		origin: origin,
	}
	if readJournal != "" {
		rj, err := os.Create(readJournal)
		if err != nil {
			return nil, err
		}
		jrw.readJournal = rj
	}
	if writerJournal != "" {
		wj, err := os.Create(writerJournal)
		if err != nil {
			return nil, err
		}
		jrw.writeJournal = wj
	}
	return
}

func (jrw *JournalReadWriter) Read(buf []byte) (n int, err error) {
	n, err = jrw.origin.Read(buf)
	if n > 0 {
		if jrw.readJournal != nil {
			_, err = jrw.readJournal.Write(buf[:n])
		}
	}
	return
}

func (jrw *JournalReadWriter) Write(buf []byte) (n int, err error) {
	n, err = jrw.origin.Write(buf)
	if err == nil {
		if jrw.writeJournal != nil {
			_, err = jrw.writeJournal.Write(buf)
		}
	}
	return
}

func (jrw *JournalReadWriter) Flush() {
	callFunc(jrw.origin, "Flush")
	if jrw.writeJournal != nil {
		_ = jrw.writeJournal.Sync()
	}
	if jrw.readJournal != nil {
		_ = jrw.readJournal.Sync()
	}
}

func (jrw *JournalReadWriter) Close() (err error) {
	err = jrw.origin.Close()
	if jrw.writeJournal != nil {
		_ = jrw.writeJournal.Sync()
		_ = jrw.writeJournal.Close()
	}
	if jrw.readJournal != nil {
		_ = jrw.readJournal.Sync()
		_ = jrw.readJournal.Close()
	}
	return
}
