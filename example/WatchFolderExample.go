package example

import (
	"fmt"
	"github.com/raomuyang/fswatcher"
	"time"
)

func onCreate(path string) {
	fmt.Println("create:", path)
}

func onRemove(filePath string) {
	fmt.Println("remove: ", filePath)
}

func onRename(filePath string) {
	fmt.Println("rename: ", filePath)
}

func onWrite(filePath string) {
	fmt.Println("write: ", filePath)
}

func test() {
	callable := fswatcher.Callable{
		OnCreate: onCreate,
		OnRename: onRename,
		OnWrite:  onWrite,
		OnRemove: onRemove,
	}

	dw, err := fswatcher.Watch("/home/user/test", callable)
	defer dw.Stop()

	if err != nil {
		panic(err)
	}
	<-time.After(time.Minute * 10)
}
