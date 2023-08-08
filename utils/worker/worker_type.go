package worker

import (
	"errors"
	"fmt"
	"strings"
)

type WorkerType int

var workers = make(map[string]WorkerType)

const (
	InvalidWorker WorkerType = iota
	HTTPWorker
	WSWorker
	GRPCWorker
	NetWorker
)

func (workertype *WorkerType) String() string {
	for k, v := range workers {
		if v == *workertype {
			return k
		}
	}
	return "unknown"
}

func (workertype *WorkerType) Set(value string) error {
	scheme := strings.SplitN(value, "://", 2)
	if len(scheme) != 2 || scheme[1] == "" {
		return fmt.Errorf("invalid worker target: %s", value)
	}
	for k, v := range workers {
		if k == scheme[0] {
			*workertype = v
			return nil
		}
	}
	return fmt.Errorf("invalid worker scheme: %s", scheme[0])
}

func AvailableWorkers() []string {
	var available []string
	for k := range workers {
		available = append(available, k)
	}
	return available
}

var (
	ErrDataLength = errors.New("data length mismatch")
	ErrHTTPStatus = errors.New("status code mismatch")
)
