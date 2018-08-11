package fswatcher

import (
	"testing"
	"os"
	"strings"
	"time"
	"sync"
)

func TestWatch(t *testing.T) {
	root := ".test"
	os.MkdirAll(root, os.ModePerm)
	defer os.RemoveAll(root)

	sub := root + "/sub"
	callable := Callable{
		OnCreate: func(filePath string) {
			t.Logf("OnCreate: %s", filePath)
			if !strings.HasPrefix(filePath, root) {
				t.Errorf("OnCreate: Unmatched file path: %s", filePath)
			}
		},
		OnRemove: func(filePath string) {
			t.Logf("OnRemove: %s", filePath)
			if !strings.HasPrefix(filePath, root) {
				t.Errorf("OnRemove: Unmatched file path: %s", filePath)
			}
		},
	}

	dw, _ := Watch(root, callable)

	f, _ := os.OpenFile(sub, os.O_CREATE|os.O_RDWR, 0777)
	f.Close()
	dw.Stop()
	os.Remove(sub)
	dw.Stop()
}

type MockCall struct {
	p     string
	done  bool
	get   string
	mutex *sync.Mutex
}

func (mock *MockCall) doAction(filePath string)  {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()
	mock.done = true
	mock.get = filePath
}

func TestDelayTrigger_AsyncDo(t *testing.T) {
	p := "test"
	mock := &MockCall{
		p: p,
		mutex: &sync.Mutex{},
	}
	trigger := NewDelayTrigger(p, 1, mock.doAction)
	trigger.AsyncDo()

	mock.mutex.Lock()
	if mock.done {
		t.Error("not delay")
	}
	mock.mutex.Unlock()

	<-time.After(time.Second + time.Millisecond * 10)

	mock.mutex.Lock()
	if !mock.done {
		t.Error("callback failed")
	}
	if strings.Compare(p, mock.get) != 0 {
		t.Error("not compare")
	}
	mock.mutex.Unlock()

}

func TestDelayTrigger_Interrupt(t *testing.T) {
	p := "test"
	done := false
	trigger := NewDelayTrigger(p, 1, func(filePath string) {
		if strings.Compare(p, filePath) != 0 {
			t.Error("not match")
		}
		done = true
	})
	trigger.timeout = time.Second
	trigger.AsyncDo()
	if done {
		t.Error("failed: not delay")
	}
	trigger.Interrupt()
	<-time.After(time.Second)
	if done {
		t.Error("interrupt failed")
	}
}