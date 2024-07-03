package worker

import (
	"sync"

	"go.f0o.dev/netbench/utils/logger"
)

var (
	syncWork_mutex   sync.Mutex
	syncWork_wait    int
	syncWork_signals map[int]chan struct{} = make(map[int]chan struct{})
)

func syncWorkAdd() int {
	syncWork_mutex.Lock()
	s := len(syncWork_signals)
	if syncWork_signals[s] == nil {
		logger.Tracew("adding sync worker", "Worker", s)
		syncWork_signals[s] = make(chan struct{}, 1)
	} else {
		logger.Fatalw("worker already exists", "Worker", s)
	}
	syncWork_mutex.Unlock()
	return s
}

func syncWorkDel(s int) {
	syncWork_mutex.Lock()
	if syncWork_signals[s] == nil {
		logger.Fatalw("worker does not exist", "Worker", s)
	}
	logger.Tracew("deleting sync worker", "Worker", s)
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
			v <- struct{}{}
		}
		syncWork_wait = 0
	}
	w := syncWork_signals[s]
	syncWork_mutex.Unlock()
	<-w
}
