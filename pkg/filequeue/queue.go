package filequeue

import (
	"fmt"
	"os"
	"sort"
	"time"
)

var baseDir = os.TempDir() + "/scanner-tool-queue"

type Queue struct {
	name    string
	counter int
}

func NewQueue(name string) *Queue {
	return &Queue{
		name: name,
	}
}

func (q *Queue) Enqueue(data []byte) error {
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

func (q *Queue) EnqueueFilePath(existingFilePath string) error {
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

func (q *Queue) Dequeue() (*os.File, error) {
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

	return file, nil
}
