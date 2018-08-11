package example

import (
	"fmt"
	"github.com/raomuyang/fswatcher"
	"time"
)

func testWatchFile() {
	callable := fswatcher.Callable{
		OnWrite: func(filePath string) {
			fmt.Println("write: ", filePath)
		},
		OnRemove: func(filePath string) {
			fmt.Println("remove: ", filePath)
		},
	}
	watcher := fswatcher.Watcher{
		Path:     "/path/to/file",
		Callable: callable}
	watcher.Watch()
	<-time.After(time.Minute)
	watcher.Stop()
}
