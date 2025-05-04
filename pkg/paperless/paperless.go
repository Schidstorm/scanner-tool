package paperless

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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
	Results []tagResult `json:"results"`
}

func (p *Paperless) createTagIfNotExist(tag string) (int, error) {
	res, err := p.httpClient.Get(p.baseUrl + tagsUrl)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != http.StatusOK {
		if res.Body != nil {
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			return 0, fmt.Errorf("bad status: %s, body: %s", res.Status, string(body))
		}

		return 0, fmt.Errorf("bad status: %s", res.Status)
	}
	defer res.Body.Close()
	var tagsResult getTagsResult
	err = json.NewDecoder(res.Body).Decode(&tagsResult)
	if err != nil {
		return 0, err
	}
	for _, tagResult := range tagsResult.Results {
		if tagResult.ID == 0 {
			continue
		}
		if tagResult.Name == tag {
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
