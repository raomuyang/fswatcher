package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/raomuyang/fswatcher"
	"github.com/raomuyang/fswatcher/filesync"
	"io/ioutil"
	"os/signal"
	"sync"
	"syscall"
)

type Config struct {
	Access        filesync.Access `yaml:"access"`
	StoreType     string          `yaml:"store_type"`
	LogPath       string          `yaml:"log_path"`

	// DEBUG(5) INFO(4) WARN(3) ERROR(2) FATAL(1) PANIC(0)
	LogLevel      uint32          `yaml:"log_level"`
	IncludeHidden bool            `yaml:"include_hidden"`
	ScanAtStart   bool            `yaml:"scan_at_start"`

	// Delay to process during a write/crate event
	OptDelay	  int 		 	  `yaml:"opt_delay"`
}

const (
	FOLDER      = ".iusync"
	DefaultConf = FOLDER + "/conf.yml"
	LogName     = "iusync.log"
)

var (
	LogDir   string
	LogLevel log.Level
	userHome string
	config 	 Config

	exit     		   = make(chan bool)
	mutex 	 		   = sync.Mutex{}
	// 待上传的队列
	postQueue 		   = make(chan string, 1000)
	// 待删除的队列
	deleteQueue 	   = make(chan string, 1000)
	// 延迟触发写操作回调
	delayWriteTriggers = make(map[string]*fswatcher.DelayTrigger)

)

func init() {
	u, err := user.Current()
	if err != nil {
		fmt.Printf("Error: cannot get current user, cause: %s\n", err)
		LogDir = FOLDER
	} else {
		userHome = u.HomeDir
		LogDir = filepath.Join(u.HomeDir, FOLDER)
	}

}

func main() {

	var target *string
	target, config = parseArgs()

	initLogger()

	printStatus()

	fsSync, err := createSyncTool()

	if err != nil {
		fmt.Printf("Error: %s", err)
		os.Exit(1)
	}

	root := path.Clean(*target)

	scanFolder(root)

	dw := startWatcher(root)

	go process(fsSync, root)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		sig := <-sigs
		log.Infof("Got signal: %v", sig)
		exit <- true
	}()

	<-exit
	log.Info("Stop watch.")
	dw.Stop()

}

// dequeue and upload/delete
func process(fsSync filesync.FileSync, root string) {
	for {
		select {
		case filePath := <-postQueue:
			upload(fsSync, filePath, root)
		case filePath := <-deleteQueue:
			deleteKey(fsSync, filePath, root)
		}
	}
}

// start watch a folder
func startWatcher(root string) fswatcher.DeepWatch {

	// RENAME之后，这个文件相当于被删除，执行REMOVE相同的操作，删除云端文件
	callable := fswatcher.Callable{
		OnRename: onDeleteAction,
		OnRemove: onDeleteAction,
		OnCreate: onCreateOrWriteAction,
		OnWrite:  onCreateOrWriteAction,
	}
	dw, err := fswatcher.Watch(root, callable)
	if err != nil {
		log.Errorf("Create watcher failed: %s", err)
		os.Exit(1)
	}
	return dw

}

func printStatus() {
	fmt.Printf("=== Start file wathcer ===\n"+
		" storage type: %s\n"+
		" log level:    %s\n"+
		" log file:     %s\n"+
		"==========================\n\n",
		config.StoreType, LogLevel.String(), filepath.Join(LogDir, LogName))
}

// init FileSync by store type
func createSyncTool() (fsSync filesync.FileSync, err error) {
	if config.StoreType == "oss" {
		fsSync = filesync.NewOSSFileSync(config.Access)
	} else if config.StoreType == "qiniu" {
		fsSync = filesync.NewQiniuFileSync(config.Access)
	} else {
		err = errors.New("unsupported store type: " + config.StoreType)
		return
	}
	return
}

// parse args
func parseArgs() (target *string, config Config) {
	target = flag.String("target", "",
		"Target path to listen (auto created when it does not exist)")
	confPath := flag.String("conf", "conf.yml", "config file path")
	help := flag.Bool("help", false, "Print usage")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if target == nil {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error: get work directory failed, cause: %s", err)
			os.Exit(1)
		}
		target = &wd
	}

	config, err := readConfig(*confPath)
	if err != nil {
		fmt.Printf("Error: Read config failed: %s, cause: %s", *confPath, err)
		os.Exit(1)
	}

	return
}

func initLogger() {
	LogLevel = log.Level(config.LogLevel)
	log.SetLevel(LogLevel)
	if len(config.LogPath) > 0 {
		LogDir = config.LogPath
	}
	prepareLogger()
}

func readConfig(path string) (config Config, err error) {
	_, err = os.Stat(path)
	if err != nil {
		newPath := filepath.Join(userHome, DefaultConf)
		fmt.Printf("[NOTE] %s not found, use default config: %s\n", path, newPath)
		path = newPath
	}

	file, err := os.Open(path)
	if err != nil {
		log.Errorf("Open %s failed", path)
		err = errors.New(fmt.Sprintf("Can not found config path in ~/%s or ./conf.yml",
			DefaultConf))
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var fileBytes []byte

	buf := make([]byte, 1024)
	for {
		n, e := reader.Read(buf)
		if e != nil {
			if e == io.EOF {
				break
			}
			err = e
			return
		}
		if n <= 0 {
			break
		}
		fileBytes = append(fileBytes, buf[:n]...)
	}

	err = yaml.Unmarshal(fileBytes, &config)
	return
}

// 上传需要一定的延迟时间，是因为文件拷贝和移动的机制不同：
// copy:  CREATE -> WRITE -> CHMOD (可能会失败)
// mv:  CREATE -> RENAME -> CHMOD
func onCreateOrWriteAction(filePath string) {

	trigger := fswatcher.NewDelayTrigger(filePath, config.OptDelay, func(filePath string) {
		log.Debugf("New file enqueue: %s", filePath)
		postQueue <- filePath
	})

	mutex.Lock()
	defer mutex.Unlock()
	lastTrigger := delayWriteTriggers[filePath]
	if lastTrigger != nil {
		lastTrigger.Interrupt()
	}
	delayWriteTriggers[filePath] = &trigger
	trigger.AsyncDo()

}

func onDeleteAction(filePath string) {
	log.Debugf("Deleted file enqueue: %s", filePath)
	deleteQueue <- filePath
}

// upload file to remote
func upload(fsSync filesync.FileSync, filePath string, root string) {
	file, err := os.Stat(filePath)
	if err != nil {
		log.Warnf("Get file info failed, cause: %s", err.Error())
		return
	}
	if file.IsDir() {
		log.Info("Skip folder upload")
	} else {
		key := getKey(filePath, root)
		url, err := fsSync.Put(filePath, key)
		if err != nil {
			log.Errorf("Error: file sync failed: %s, cause: %s", filePath, err.Error())
		} else {
			fmt.Println("New:", url)
		}
	}
}

// delete from remote storage
func deleteKey(fsSync filesync.FileSync, filePath string, root string) {
	key := getKey(filePath, root)
	err := fsSync.Delete(key)
	log.Infof("Delete key: %s, err: %v", key, err)
}

func getKey(filePath, root string) string {
	key := strings.Replace(filePath, root, "", 1)
	if key[0] == '/' {
		key = key[1:]
	}
	return key
}

func prepareLogger() {
	_, err := os.Stat(LogDir)
	if err != nil {
		err = os.MkdirAll(LogDir, os.ModePerm)
		if err != nil {
			fmt.Printf("Error: init logger failed, cause: %s", err)
			os.Exit(1)
		}
	}
	filePath := filepath.Join(LogDir, LogName)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0766)
	if err != nil {
		fmt.Printf("Error: create log file failed, cause: %s", err)
		os.Exit(1)
	}
	log.SetOutput(file)
}

// 初始化扫描文件夹，将所有文件添加到待上传的队列中
func scanFolder(root string) {
	if !config.ScanAtStart {
		log.Info("Scan at start: false")
		return
	}

	files, err := ioutil.ReadDir(root)
	if err != nil {
		log.Warnf("read directory error: %s", err.Error())
		return
	}

	for i := range files {
		file := files[i]
		sub := filepath.Join(root, file.Name())
		if file.Name()[0] == '.' && !config.IncludeHidden {
			log.Infof("Ignore hidden file/path: %s", sub)
			continue
		}

		if file.IsDir() {
			scanFolder(sub)
		} else {
			postQueue <- sub
		}
	}
}
