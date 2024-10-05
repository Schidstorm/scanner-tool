package cifs

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/hirochachacha/go-smb2"
	"github.com/sirupsen/logrus"
)

var filePrefix = "zzzzzzz-scan-"

type Options struct {
	Username string
	Password string
	Hostname string
	Port     int
	Share    string
	BasePath string
}

type Cifs struct {
	options        Options
	connection     net.Conn
	session        *smb2.Session
	share          *smb2.Share
	cifsOpened     bool
	cifsAccessLock sync.Mutex
	closeChannel   chan struct{}
	wgClosed       *sync.WaitGroup
}

var cifsOpenRetryDelay = 2 * time.Second
var cifsCheckConnectionDelay = 30 * time.Second

func NewCifs(opts Options) *Cifs {
	if opts.Port == 0 {
		opts.Port = 445
	}

	wgClosed := new(sync.WaitGroup)
	wgClosed.Add(1)

	return &Cifs{
		options:      opts,
		closeChannel: make(chan struct{}),
		wgClosed:     wgClosed,
	}
}

func (c *Cifs) Start() error {
	c.beginEnsureCifsOpen()
	return nil
}

func (c *Cifs) Stop() error {
	close(c.closeChannel)
	c.wgClosed.Wait()
	return c.Close()
}

func (c *Cifs) beginEnsureCifsOpen() {
	go func() {
		defer c.wgClosed.Done()
		for {
			{
				c.cifsAccessLock.Lock()

				if c.cifsOpened {
					_, err := c.share.ReadDir(c.options.BasePath)
					if err != nil {
						logrus.WithError(err).Warn("CIFS connection lost")
						c.closeNoLock()
						c.cifsOpened = false
					} else {
						logrus.Info("CIFS connection is open")
						c.cifsAccessLock.Unlock()
						if c.waitOrClose(cifsCheckConnectionDelay) {
							return
						}
						continue
					}
				}

				c.cifsAccessLock.Unlock()
			}

			for {
				logrus.Info("Opening CIFS connection")
				err := c.openSingle()
				if err == nil {
					logrus.Info("Opened CIFS connection")
					c.cifsAccessLock.Lock()
					c.cifsOpened = true
					c.cifsAccessLock.Unlock()
					if c.waitOrClose(cifsCheckConnectionDelay) {
						return
					}
					break
				} else {
					logrus.WithError(err).Warn("Failed to open CIFS connection")
				}
				if c.waitOrClose(cifsOpenRetryDelay) {
					return
				}
			}
		}
	}()
}

func (c *Cifs) waitOrClose(duration time.Duration) (close bool) {
	select {
	case <-time.After(duration):
		return false
	case <-c.closeChannel:
		return true
	}
}

func (c *Cifs) openSingle() error {
	var err error
	c.connection, err = net.Dial("tcp", fmt.Sprintf("%s:%d", c.options.Hostname, c.options.Port))
	if err != nil {
		return err
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     c.options.Username,
			Password: c.options.Password,
		},
	}

	c.session, err = d.Dial(c.connection)
	if err != nil {
		c.closeNoLock()
		return err
	}

	c.share, err = c.session.Mount(c.options.Share)
	if err != nil {
		c.closeNoLock()
		return err
	}

	return nil
}

func (c *Cifs) Close() error {
	c.cifsAccessLock.Lock()
	defer c.cifsAccessLock.Unlock()
	return c.closeNoLock()
}

func (c *Cifs) closeNoLock() error {
	if c.share != nil {
		c.share.Umount()
	}
	if c.session != nil {
		c.session.Logoff()
	}
	if c.connection != nil {
		c.connection.Close()
	}
	return nil
}

func (c *Cifs) Upload(p string, data []byte) error {
	return c.accessShare(func(share *smb2.Share) error {
		logrus.WithField("path", p).Info("Uploading file")
		f, err := share.Create(path.Join(c.options.BasePath, p))
		if err != nil {
			return err
		}
		defer f.Close()
		f.Write(data)
		return err
	})
}

func (c *Cifs) Download(p string) ([]byte, error) {
	// check not below base path
	if path.Clean(p) != p {
		return nil, fmt.Errorf("invalid path")
	}

	var result []byte
	err := c.accessShare(func(share *smb2.Share) error {
		logrus.WithField("path", p).Info("Downloading file")
		f, err := share.Open(path.Join(c.options.BasePath, p))
		if err != nil {
			return err
		}
		defer f.Close()
		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		result = b
		return nil
	})

	return result, err
}

func (c *Cifs) List() ([]string, error) {
	var result []string
	err := c.accessShare(func(share *smb2.Share) error {
		logrus.Info("Listing files")
		files, err := share.ReadDir(c.options.BasePath)
		if err != nil {
			return err
		}

		for _, file := range files {
			result = append(result, file.Name())
		}

		return nil
	})

	return result, err
}

// Delete
func (c *Cifs) Delete(paths ...string) error {
	return c.accessShare(func(share *smb2.Share) error {
		logrus.WithField("paths", paths).Info("Deleting files")
		var errors []error
		for _, p := range paths {
			err := share.Remove(path.Join(c.options.BasePath, p))
			if err != nil {
				logrus.WithError(err).Warn("Failed to delete file")
				errors = append(errors, err)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("failed to delete files")
		}

		return nil
	})
}

func (c *Cifs) accessShare(handler func(share *smb2.Share) error) error {
	for {
		c.cifsAccessLock.Lock()
		defer c.cifsAccessLock.Unlock()
		if c.cifsOpened {
			return handler(c.share)
		}
		time.Sleep(cifsOpenRetryDelay)
	}
}

func (c *Cifs) SwapFileNames(nameA, nameB string) error {
	return c.accessShare(func(share *smb2.Share) error {
		logrus.WithFields(logrus.Fields{
			"nameA": nameA,
			"nameB": nameB,
		}).Info("Swapping file names")

		c.share.Rename(path.Join(c.options.BasePath, nameA), path.Join(c.options.BasePath, "tmp-"+nameA))
		c.share.Rename(path.Join(c.options.BasePath, nameB), path.Join(c.options.BasePath, nameA))
		c.share.Rename(path.Join(c.options.BasePath, "tmp-"+nameA), path.Join(c.options.BasePath, nameB))

		return nil
	})
}

func (c *Cifs) RenameFile(oldName, newName string) error {
	return c.accessShare(func(share *smb2.Share) error {
		logrus.WithFields(logrus.Fields{
			"oldName": oldName,
			"newName": newName,
		}).Info("Renaming file")

		return c.share.Rename(path.Join(c.options.BasePath, oldName), path.Join(c.options.BasePath, newName))
	})
}

func (c *Cifs) NextFileId() (int, error) {

	files, err := c.List()
	if err != nil {
		return 0, err
	}

	fileNames := map[string]struct{}{}
	for _, f := range files {
		fileNames[f] = struct{}{}
	}

	for i := 0; ; i++ {
		pngFileName := fmt.Sprintf("%s%02d%s", filePrefix, i, ".png")
		pdfFileName := fmt.Sprintf("%s%02d%s", filePrefix, i, ".pdf")

		if _, ok := fileNames[pngFileName]; !ok {
			if _, ok := fileNames[pdfFileName]; !ok {
				return i, nil
			}
		}
	}
}

func (c *Cifs) MakeUnique(p string, t time.Time) (string, error) {
	randBytes := make([]byte, 4)
	secondsU32 := uint32(t.Unix() % (2 << 32))
	binary.LittleEndian.PutUint32(randBytes, secondsU32)
	randomSuffix := hex.EncodeToString(randBytes)

	parts := strings.Split(p, ".")
	noRnd := true
	for i, part := range parts {
		if strings.HasPrefix(part, "RND") {
			parts[i] = "RND" + randomSuffix
			noRnd = false
		}
	}
	if noRnd {
		ext := parts[len(parts)-1]
		parts[len(parts)-1] = "RND" + randomSuffix + "." + ext
	}

	newPath := strings.Join(parts, ".")

	return newPath, c.RenameFile(p, newPath)
}
