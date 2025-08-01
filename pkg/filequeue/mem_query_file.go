package filequeue

import (
	"io"
	"os"
	"strconv"
)

type MemQueryFile struct {
	Name string
	Data []byte
}

/*
io.Reader
	io.ReaderAt
	io.Closer

	Done()
	Size() (int64, error)
*/

func (f MemQueryFile) Done() {

}

func (f *MemQueryFile) Size() (int64, error) {
	return int64(len(f.Data)), nil
}

func (f *MemQueryFile) Read(p []byte) (n int, err error) {
	if len(f.Data) == 0 {
		return 0, io.EOF
	}

	n = copy(p, f.Data)
	f.Data = f.Data[n:]

	return n, nil
}

func (f *MemQueryFile) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || int(off) >= len(f.Data) {
		return 0, io.EOF
	}

	if len(p) == 0 {
		return 0, nil
	}

	n = copy(p, f.Data[off:])
	if n < len(p) {
		return n, io.EOF
	}

	return n, nil
}

func (f MemQueryFile) Close() error {
	// No resources to release
	return nil
}

type MemQueryFileQueue struct {
	Files []MemQueryFile
}

func (q *MemQueryFileQueue) Enqueue(data []byte) error {
	q.Files = append(q.Files, MemQueryFile{
		Name: "file-" + strconv.FormatInt(int64(len(q.Files)), 10),
		Data: data,
	})
	return nil
}

func (q *MemQueryFileQueue) EnqueueFilePath(existingFilePath string) error {
	data, err := os.ReadFile(existingFilePath)
	if err != nil {
		return err
	}
	q.Files = append(q.Files, MemQueryFile{
		Name: existingFilePath,
		Data: data,
	})
	return nil
}

func (q *MemQueryFileQueue) Dequeue() (QueueFile, error) {
	if len(q.Files) == 0 {
		return nil, io.EOF
	}

	file := q.Files[0]
	q.Files = q.Files[1:]

	return &file, nil
}
