package fswatcher

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
	"sync"
)

// Listen for changes to files or directories.
// When the target is a directory, all subfiles and subdirectories are recursively monitored (the file at initialization is not processed).
// When the target is a file, only listen for changes to this file, and stop listening when the file is deleted.
type DeepWatch struct {
	callable    Callable
	watcherList []Watcher
}

// Stop all watchers
func (dw DeepWatch) Stop() {
	for i := range dw.watcherList {
		w := dw.watcherList[i]
		w.Stop()
	}
}

// Start watch
func Watch(path string, callable Callable) (dw DeepWatch, err error) {
	dw = DeepWatch{callable: callable}
	dw.watcherList = make([]Watcher, 0)

	fileInfo, err := os.Stat(path)
	if err != nil {
		return
	}

	if fileInfo.IsDir() {
		dw.watchFolder(path)
	} else {
		dw.watchPath(path)
	}
	return
}

func (dw *DeepWatch) watchPath(path string) {
	callable := dw.callable

	w := Watcher{
		Path:     path,
		Callable: callable,
	}

	lock.Lock()
	dw.watcherList = append(dw.watcherList, w)
	lock.Unlock()

	w.Watch()
}

func (dw *DeepWatch) watchFolder(path string) (err error) {
	dw.watchPath(path)

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}

	for i := range files {
		file := files[i]
		sub := filepath.Join(path, file.Name())
		if file.IsDir() {
			subWatchErr := dw.watchFolder(sub)
			if subWatchErr != nil {
				log.Errorf("Error: watch sub folder exception: %s", err)
			}
		} else {
			log.Debugf("Ignore initial file: %s", sub)
		}
	}

	return nil
}

func (dw *DeepWatch) onCreateFunc() func(path string) {
	return func(path string) {
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Warnf("Get file stat failed: %s", path)
		} else if fileInfo.IsDir() {
			dw.watchFolder(path)
		}

		dw.callable.doOnCreate(path)
	}
}

// 通知到达时延迟执行，可以被打断
type DelayTrigger struct {
	filePath  string
	callback  func(filePath string)
	interrupt chan int
	timeout   time.Duration
	started   bool
	mutex     *sync.Mutex
}

const defaultDelay = 3 // seconds
func NewDelayTrigger(filePath string, delay int, callback func(filePath string)) DelayTrigger {
	if delay == 0 {
		delay = defaultDelay
	}
	mutex := &sync.Mutex{}
	return DelayTrigger{
		filePath:  filePath,
		callback:  callback,
		interrupt: make(chan int),
		timeout:   time.Second * time.Duration(delay),
		mutex:     mutex,
	}
}

// 中断即将执行的动作
func (trigger DelayTrigger) Interrupt() {
	select {
	case trigger.interrupt <- 1:
	case <-time.After(time.Second):
	}
}

// 异步执行
func (trigger DelayTrigger) AsyncDo() {

	trigger.mutex.Lock()
	defer trigger.mutex.Unlock()
	if trigger.started {
		return
	}

	trigger.started = true

	go func(){

		select {
		case i := <-trigger.interrupt:
			if i > 0 {
				return
			}

		case <-time.After(trigger.timeout):
			if trigger.callback != nil {
				log.Debugf("callback: %s", trigger.filePath)
				trigger.mutex.Lock()
				defer trigger.mutex.Unlock()
				trigger.callback(trigger.filePath)
			}
		}
	}()
}
