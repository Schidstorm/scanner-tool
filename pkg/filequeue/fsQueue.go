package filequeue

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/schidstorm/scanner-tool/pkg/logger"
)

var log = logger.Logger(FsQueue{})

type fsQueueFile struct {
	*os.File
}

func (f *fsQueueFile) Done() {
	os.Remove(f.Name())
}

func (f *fsQueueFile) Size() (int64, error) {
	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

type FsQueue struct {
	name    string
	counter int
}

func NewFsQueue(name string) *FsQueue {
	return &FsQueue{
		name: name,
	}
}

func (q *FsQueue) Enqueue(data []byte) error {
	log.Debugf("Enqueueing data to queue %s", q.name)
	dir := baseDir + "/" + q.name
	err := ensureDir(dir)
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/queue-%d-%d", dir, time.Now().Unix(), q.counter)
	q.counter++
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (q *FsQueue) EnqueueFilePath(existingFilePath string) error {
	log.Debugf("Enqueueing file to queue %s", q.name)
	dir := baseDir + "/" + q.name
	err := ensureDir(dir)
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/queue-%d-%d", dir, time.Now().Unix(), q.counter)
	q.counter++
	err = os.Rename(existingFilePath, filePath)
	if err != nil {
		return err
	}

	return nil
}

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func (q *FsQueue) Dequeue() (QueueFile, error) {
	log.Debugf("Dequeueing file from queue %s", q.name)
	dir := baseDir + "/" + q.name
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	filPaths, err := waitUntilSomeFiles(dir)
	if err != nil || len(filPaths) == 0 {
		return nil, err
	}

	file, err := os.Open(filPaths[0])
	if err != nil {
		return nil, err
	}

	return &fsQueueFile{File: file}, nil
}

func waitUntilSomeFiles(dir string) ([]string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	barrier := make(chan struct{})
	defer close(barrier)

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				log.Debugf("event: %v", event)
				if event.Has(fsnotify.Create) {
					log.Debugf("file created: %s", event.Name)
					barrier <- struct{}{}
					return
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				log.WithError(err).Error("error")
			}
		}
	}()

	err = watcher.Add(dir)
	if err != nil {
		return nil, err
	}

	files := listFiles(dir)
	if len(files) > 0 {
		return files, nil
	}

	<-barrier

	files = listFiles(dir)
	if len(files) > 0 {
		return files, nil
	}

	return nil, nil
}

func listFiles(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.WithError(err).Warn("Failed to list files")
		return nil
	}

	filPaths := make([]string, 0, len(files))
	for _, file := range files {
		filPaths = append(filPaths, dir+"/"+file.Name())
	}
	sort.Strings(filPaths)

	return filPaths
}
