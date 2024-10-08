package server

type DocumentGroup struct {
	Name  string
	Paths DocumentPathList
}

type DocumentGroupList []DocumentGroup

func (d DocumentGroupList) Len() int {
	return len(d)
}

func (d DocumentGroupList) Less(i, j int) bool {
	return d[i].Name < d[j].Name
}

func (d DocumentGroupList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type DocumentPath struct {
	Pdf   string
	Image string
}

type DocumentPathList []DocumentPath

func (d DocumentPathList) Len() int {
	return len(d)
}

func (d DocumentPathList) Less(i, j int) bool {
	if d[i].Pdf == d[j].Pdf {
		return d[i].Image < d[j].Image
	}

	return d[i].Pdf < d[j].Pdf
}

func (d DocumentPathList) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
