package main

import "sync"

type Counter struct {
	Count  int64
	Locker sync.RWMutex
}
