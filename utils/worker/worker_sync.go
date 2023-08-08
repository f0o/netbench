package worker

import (
	"sync"

	"go.f0o.dev/netbench/utils/logger"
)

var syncWork_mutex sync.Mutex
var syncWork_wait int
var syncWork_signals map[int]chan bool = make(map[int]chan bool)

func syncWorkAdd() int {
	syncWork_mutex.Lock()
	s := len(syncWork_signals)
	if syncWork_signals[s] == nil {
		logger.Debugw("adding sync worker", "Worker", s)
		syncWork_signals[s] = make(chan bool, 1)
	}
	syncWork_mutex.Unlock()
	return s
}

func syncWorkDel(s int) {
	syncWork_mutex.Lock()
	if syncWork_signals[s] == nil {
		logger.Fatalw("worker does not exist", "Worker", s)
	}
	logger.Debugw("deleting sync worker", "Worker", s)
	close(syncWork_signals[s])
	delete(syncWork_signals, s)
	syncWork_mutex.Unlock()
}

func syncWorkWait(s int) {
	syncWork_mutex.Lock()
	if syncWork_signals[s] == nil {
		logger.Fatalw("worker does not exist", "Worker", s)
	}
	syncWork_wait++
	if syncWork_wait == len(syncWork_signals) {
		for _, v := range syncWork_signals {
			v <- true
		}
		syncWork_wait = 0
	}
	syncWork_mutex.Unlock()
	<-syncWork_signals[s]
}
