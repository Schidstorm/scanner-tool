package scan

type Scanner interface {
	Scan() ([]string, error)
}
