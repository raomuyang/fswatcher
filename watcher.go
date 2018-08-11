package fswatcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Callable struct {
	// CREATE corresponding function
	OnCreate func(filePath string)
	// REMOVE corresponding function
	OnRemove func(filePath string)
	// WRITE corresponding function
	OnWrite func(filePath string)
	// RENAME corresponding function
	OnRename func(filePath string)
}

func (callable Callable) doOnCreate(filePath string)  {
	if callable.OnCreate != nil {
		callable.OnCreate(filePath)
	}
}

func (callable Callable) doOnRemove(filePath string)  {
	if callable.OnRemove != nil {
		callable.OnRemove(filePath)
	}
}

func (callable Callable) doOnWrite(filePath string)  {
	if callable.OnWrite != nil {
		callable.OnWrite(filePath)
	}
}

func (callable Callable) doOnRename(filePath string)  {
	if callable.OnRename != nil {
		callable.OnRename(filePath)
	}
}

// The watcher of file/folder, listen to the changes of file or subitems
type Watcher struct {
	// target path to watch
	Path      string
	Callable  Callable
	stop      chan bool
	fsWatcher *fsnotify.Watcher
}

var lock sync.Mutex

// Start the watch process in a new goroutine
func (watcher *Watcher) Watch() error {
	lock.Lock()
	if watcher.stop != nil {
		return errors.New("already started")
	}
	lock.Unlock()

	watcher.stop = make(chan bool)
	fsWatcher, err := fsnotify.NewWatcher()
	watcher.fsWatcher = fsWatcher

	if err != nil {
		log.Fatalf("error: %s", err)
		return err
	}

	log.Infof("Start watcher: %s", watcher.Path)
	go func() {
		for {
			select {
			case event, ok := <-fsWatcher.Events:
				if !ok {
					break
				}
				watcher.onEvent(event)
			case err = <-fsWatcher.Errors:
				if err != nil {
					log.Warnf("fsnotify error: %s", err.Error())
				}
			case stop := <-watcher.stop:
				if stop {
					watcher.stop = nil
					break
				}
			}
		}
	}()
	return fsWatcher.Add(watcher.Path)
}

// Stop the watch process
func (watcher Watcher) Stop() {
	select {
	case watcher.stop <- true:
		log.Infof("Watcher closed: %s", watcher.Path)
	case <-time.After(time.Second):
		log.Infof("Watcher closed (second): %s", watcher.Path)
	}
}

func (watcher Watcher) removeWatch(filePath string)  {
	log.Debugf("Remove watch for %s", filePath)
	watcher.fsWatcher.Remove(filePath)

	if filePath == watcher.Path {
		watcher.Stop()
	}
}

// see fsnotify: func (op Op) String() string
func (watcher Watcher) onEvent(event fsnotify.Event) {

	log.Debugf("event: %v", event)

	if event.Op&fsnotify.Write == fsnotify.Write {
		watcher.Callable.doOnWrite(event.Name)
	}

	if event.Op&fsnotify.Create == fsnotify.Create {
		watcher.Callable.doOnCreate(event.Name)
	}

	if event.Op&fsnotify.Rename == fsnotify.Rename {
		// RENAME or REMOVE|RENAME (folder)
		log.Infof("Rename: %s", watcher.Path)
		watcher.removeWatch(event.Name)
		watcher.Callable.doOnRename(event.Name)
	} else if event.Op&fsnotify.Remove == fsnotify.Remove {
		log.Infof("Remove: %s", watcher.Path)
		watcher.removeWatch(event.Name)
		watcher.Callable.doOnRemove(event.Name)
	}
}
