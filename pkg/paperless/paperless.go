package paperless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var httpClientTimeout = 1 * time.Hour
var postDocumentUrl = "/api/documents/post_document/"
var tagsUrl = "/api/tags/"

type addHeaderTransport struct {
	T       http.RoundTripper
	Headers map[string]string
}

func (adt *addHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range adt.Headers {
		req.Header.Add(key, value)
	}
	return adt.T.RoundTrip(req)
}

type Paperless struct {
	token      string
	baseUrl    string
	httpClient *http.Client
}

type UploadOptions struct {
	Title               string
	Created             *time.Time
	Correspondent       *string
	DocumentType        *string
	StoragePath         *string
	Tags                []string
	ArchiveSerialNumber *string
}

func NewPaperless(baseUrl, token string) *Paperless {
	httpClient := &http.Client{
		Timeout: httpClientTimeout,
		Transport: &addHeaderTransport{
			T: http.DefaultTransport,
			Headers: map[string]string{
				"Authorization": "Token " + token,
			},
		},
	}

	if baseUrl[len(baseUrl)-1] == '/' {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}

	return &Paperless{
		baseUrl:    baseUrl,
		token:      token,
		httpClient: httpClient,
	}
}

func (p *Paperless) Upload(file io.Reader, options UploadOptions) error {
	if options.Title == "" {
		return fmt.Errorf("title is required")
	}

	var tagIds []int
	for _, tag := range options.Tags {
		tagId, err := p.createTagIfNotExist(tag)
		if err != nil {
			return fmt.Errorf("failed to create tag %s: %v", tag, err)
		}
		tagIds = append(tagIds, tagId)
	}

	var multipartBuffer bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBuffer)

	var errs []error
	errs = append(errs, addField(multipartWriter, "title", options.Title))
	if options.Created != nil {
		errs = append(errs, addField(multipartWriter, "created", options.Created.Format(time.RFC3339)))
	}
	if options.Correspondent != nil {
		errs = append(errs, addField(multipartWriter, "correspondent", *options.Correspondent))
	}
	if options.DocumentType != nil {
		errs = append(errs, addField(multipartWriter, "document_type", *options.DocumentType))
	}
	if options.StoragePath != nil {
		errs = append(errs, addField(multipartWriter, "storage_path", *options.StoragePath))

	}
	if options.ArchiveSerialNumber != nil {
		errs = append(errs, addField(multipartWriter, "archive_serial_number", *options.ArchiveSerialNumber))
	}
	if len(options.Tags) > 0 {
		for _, tag := range tagIds {
			errs = append(errs, addField(multipartWriter, "tags", strconv.Itoa(tag)))
		}
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes("document"), escapeQuotes(options.Title)))
	h.Set("Content-Type", "application/pdf")
	documentFormFile, err := multipartWriter.CreatePart(h)
	if err != nil {
		return err
	}

	_, err = io.Copy(documentFormFile, file)
	if err != nil {
		return err
	}
	multipartWriter.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", p.baseUrl+postDocumentUrl, &multipartBuffer)
	if err != nil {
		return err
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// Set the authorization header
	// req.Header.Set("Authorization", "Token "+p.token)

	// Submit the request
	res, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		if res.Body != nil {
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			return fmt.Errorf("bad status: %s, body: %s", res.Status, string(body))
		}

		return fmt.Errorf("bad status: %s", res.Status)
	}

	if res.Body != nil {
		res.Body.Close()
	}

	return nil
}

type tagResult struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type getTagsResult struct {
	Next    *string     `json:"next"`
	Count   int         `json:"count"`
	Results []tagResult `json:"results"`
}

func (p *Paperless) GetTags() ([]tagResult, error) {
	u := &url.URL{
		Path: tagsUrl,
	}
	query := u.Query()
	query.Add("page_size", "50")

	var tags []tagResult
	for i := 1; ; i++ {
		var tagsResult getTagsResult

		query.Set("page", strconv.Itoa(i))
		u.RawQuery = query.Encode()
		err := p.apiCallParsed("GET", *u, nil, &tagsResult)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags: %v", err)
		}
		tags = append(tags, tagsResult.Results...)
		if tagsResult.Next == nil {
			break
		}
	}

	return tags, nil
}

func (p *Paperless) apiCallParsed(method string, url url.URL, body io.Reader, result interface{}) error {
	res, err := p.apiCall(method, url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bad status: %s, body: %s", res.Status, string(bodyBytes))
	}
	err = json.NewDecoder(res.Body).Decode(result)
	if err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}
	return nil
}

func (p *Paperless) apiCall(method string, u url.URL, body io.Reader) (*http.Response, error) {
	baseUrlUrl, _ := url.Parse(p.baseUrl)
	u.Host = baseUrlUrl.Host
	u.Scheme = baseUrlUrl.Scheme

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	return p.httpClient.Do(req)
}

func (p *Paperless) EditTag(tagID int, newName string) error {
	if tagID <= 0 {
		return fmt.Errorf("tag ID must be greater than 0")
	}
	if newName == "" {
		return fmt.Errorf("new name is required")
	}

	tagUpdateUrl := fmt.Sprintf("%s%d/", p.baseUrl+tagsUrl, tagID)
	updateData := map[string]string{"name": newName}
	data, err := json.Marshal(updateData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", tagUpdateUrl, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("bad status: %s, body: %s", res.Status, string(body))
	}

	return nil
}

func (p *Paperless) createTagIfNotExist(tag string) (int, error) {
	tags, err := p.GetTags()
	if err != nil {
		return 0, fmt.Errorf("failed to get tags: %v", err)
	}

	for _, tagResult := range tags {
		if tagResult.ID == 0 {
			continue
		}
		if strings.ToLower(tagResult.Name) == tag {
			return tagResult.ID, nil
		}
	}

	// If the tag does not exist, create it
	createTagRes, err := p.httpClient.Post(p.baseUrl+tagsUrl, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"name": "%s"}`, tag))))
	if err != nil {
		return 0, err
	}

	if createTagRes.StatusCode != http.StatusCreated {
		if createTagRes.Body != nil {
			body, _ := io.ReadAll(createTagRes.Body)
			createTagRes.Body.Close()
			return 0, fmt.Errorf("bad status: %s, body: %s", createTagRes.Status, string(body))
		}

		return 0, fmt.Errorf("bad status: %s", createTagRes.Status)
	}
	defer createTagRes.Body.Close()

	var createTagResult tagResult
	err = json.NewDecoder(createTagRes.Body).Decode(&createTagResult)
	if err != nil {
		return 0, err
	}

	if createTagResult.ID == 0 {
		return 0, fmt.Errorf("tag not created")
	}
	return createTagResult.ID, nil

}

func addField(mw *multipart.Writer, fieldname, value string) error {
	w, err := mw.CreateFormField(fieldname)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(value))
	return err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
