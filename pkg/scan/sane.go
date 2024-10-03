package scan

import (
	"fmt"
	"image/png"
	"io"
	"strconv"
	"time"

	"github.com/tjgq/sane"
)

var imageEncode = png.Encode

const deviceOpenRetries = 10
const deviceOpenRetryDelay = 100 * time.Millisecond

type Options struct {
	SaneOptions []string
}

type SaneScanner struct {
	deviceName string
	options    Options
}

func NewScanner(opts Options) *SaneScanner {
	return &SaneScanner{
		options: opts,
	}
}

func (s *SaneScanner) Scan(outputImage io.Writer) error {
	c, err := s.retryOpenDevice()
	if err != nil {
		return err
	}
	defer c.Close()

	if err := parseOptions(c, s.options.SaneOptions); err != nil {
		return err
	}

	img, err := c.ReadImage()
	if err != nil {
		return err
	}

	if err := imageEncode(outputImage, img); err != nil {
		return err
	}

	return nil
}

func (s *SaneScanner) rescanDeviceName() error {
	devs, err := listDevices()
	if err != nil {
		return err
	}
	if len(devs) == 0 {
		return fmt.Errorf("no devices available")
	}
	s.deviceName = devs[0]
	return nil
}

func (s *SaneScanner) getDeviceName() (string, error) {
	if s.deviceName == "" {
		err := s.rescanDeviceName()
		if err != nil {
			return "", err
		}
	}
	return s.deviceName, nil
}

func (s *SaneScanner) retryOpenDevice() (*sane.Conn, error) {
	var c *sane.Conn

	err := retry(func() error {
		deviceName, err := s.getDeviceName()
		if err != nil {
			return err
		}

		c, err = sane.Open(deviceName)
		if err != nil {
			s.rescanDeviceName()
		}

		return err
	}, deviceOpenRetries, deviceOpenRetryDelay)

	return c, err
}

func retry(f func() error, retries int, delay time.Duration) error {
	for i := 0; i < retries; i++ {
		if err := f(); err == nil {
			return nil
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("failed after %d tries", retries)
}

func findOption(opts []sane.Option, name string) (*sane.Option, error) {
	for _, o := range opts {
		if o.Name == name {
			return &o, nil
		}
	}
	return nil, fmt.Errorf("no such option")
}

func parseBool(s string) (interface{}, error) {
	if s == "yes" || s == "true" || s == "1" {
		return true, nil
	}
	if s == "no" || s == "false" || s == "0" {
		return false, nil
	}
	return nil, fmt.Errorf("not a boolean value")
}

func parseOptions(c *sane.Conn, args []string) error {
	invalidArg := fmt.Errorf("invalid argument")
	if len(args)%2 != 0 {
		return invalidArg // expect option/value pairs
	}
	for i := 0; i < len(args); i += 2 {
		if args[i][0] != '-' || args[i+1][0] == '-' {
			return invalidArg
		}
		o, err := findOption(c.Options(), args[i][1:])
		if err != nil {
			return invalidArg // no such option
		}
		var v interface{}
		if o.IsAutomatic && args[i+1] == "auto" {
			v = sane.Auto // set to auto value
		} else {
			switch o.Type {
			case sane.TypeBool:
				if v, err = parseBool(args[i+1]); err != nil {
					return invalidArg // not a bool
				}
			case sane.TypeInt:
				if v, err = strconv.Atoi(args[i+1]); err != nil {
					return invalidArg // not an int
				}
			case sane.TypeFloat:
				if v, err = strconv.ParseFloat(args[i+1], 64); err != nil {
					return invalidArg // not a float
				}
			case sane.TypeString:
				v = args[i+1]
			}
		}
		if _, err := c.SetOption(o.Name, v); err != nil {
			return err // can't set option
		}
	}
	return nil
}

func listDevices() ([]string, error) {
	devs, err := sane.Devices()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, d := range devs {
		names = append(names, d.Name)
	}
	return names, nil
}
