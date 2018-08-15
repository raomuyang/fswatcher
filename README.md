# FSWATCHER

[![Build Status](https://travis-ci.org/raomuyang/fswatcher.svg?branch=master)](https://travis-ci.org/raomuyang/fswatcher) 
[![Go Report Card](https://goreportcard.com/badge/github.com/raomuyang/fswatcher)](https://goreportcard.com/report/github.com/raomuyang/fswatcher)
[![GoDoc](https://godoc.org/github.com/raomuyang/fswatcher?status.svg)](https://godoc.org/github.com/raomuyang/fswatcher)

A library to listen to the changes of File/Folder：

* [x] CREATE
* [x] REMOVE
* [x] RENAME
* [x] DELETE

## Installation

```shell
go get github.com/raomuyang/fswatcher/iusync
```

## Watcher

Listen for changes to files or directories.
* Directory: All subfiles and subdirectories are recursively monitored (the file at initialization is not processed).
* File: only listen for changes to this file, and stop listening when the file is deleted.

```go
package exapmle
import (
	"github.com/raomuyang/fswatcher"
	"time"
)

func demo() {
	callable := fswatcher.Callable{
		OnCreate: onCreateFunc,
		OnRename: onRenameFunc,
		OnWrite:  onWriteFunc,
		OnRemove: onRemoveFunc,
	}

	dw, err := fswatcher.Watch("/path/to/target/", callable)
	defer dw.Stop()

	if err != nil {
		panic(err)
	}
	
	<-time.After(time.Minute * 10)
}
```

## iusync

A tool for synchronizing local files to cloud storage in real time. 
When you use Markdown to write a blog, you can easily sync the local image of the 
image to the map bed and get the image link.

### Usage

* install iusync by golang (You can also download the compiled executable binary file from release.)
```shell
go get github.com/raomuyang/fswatcher/iusync
```

* write config

```yaml
store_type: qiniu # qiniu / oss
log_level: 5 # DEBUG(5) INFO(4) WARN(3) ERROR(2) FATAL(1) PANIC(0)
log_path: /path/to/save/log # specify the log path
scan_at_start: true # default false
include_hidden: true # default false
opt_delay: 3 # default 3 (seconds)
access:
  access_key_id: your_access_key_id
  access_key_secret: your_access_key_secret
  bucket: your_bucket_name
  domain: http://eg.xxx.cn # The bound custom domain name(to assemble a visible url)
  endpoint: xxx  # if the store type is qiniu, this section is optional
```

* run

```shell
iusync -target=/path/to/target/folder/or/image [-conf=specify/conf/file]
```

## Future

* [ ] file hook
