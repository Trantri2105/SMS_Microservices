package logger

import (
	"os"
	"sync/atomic"
)

type ReopenableWriteSyncer struct {
	file string
	cur  atomic.Value
}

func NewReopenableWriteSyncer(file string) (*ReopenableWriteSyncer, error) {
	ws := &ReopenableWriteSyncer{
		file: file,
	}
	if err := ws.Reload(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (ws *ReopenableWriteSyncer) getFile() *os.File {
	return ws.cur.Load().(*os.File)
}

func (ws *ReopenableWriteSyncer) Reload() error {
	file, err := os.OpenFile(ws.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	old := ws.cur.Swap(file)
	if old != nil {
		return old.(*os.File).Close()
	}
	return nil
}

func (ws *ReopenableWriteSyncer) Sync() error {
	return ws.getFile().Sync()
}

func (ws *ReopenableWriteSyncer) Close() error {
	return ws.getFile().Close()
}

func (ws *ReopenableWriteSyncer) Write(p []byte) (n int, err error) {
	return ws.getFile().Write(p)
}
