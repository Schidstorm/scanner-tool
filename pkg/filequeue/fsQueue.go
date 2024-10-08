package filequeue

import (
	"fmt"
	"os"
	"sort"
	"time"
)

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
	dir := baseDir + "/" + q.name
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	filPaths := make([]string, 0, len(files))
	for _, file := range files {
		filPaths = append(filPaths, dir+"/"+file.Name())
	}
	sort.Strings(filPaths)

	file, err := os.Open(filPaths[0])
	if err != nil {
		return nil, err
	}

	return &fsQueueFile{File: file}, nil
}
