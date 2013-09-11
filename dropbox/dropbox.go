package dropbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	rootRegexp *regexp.Regexp = regexp.MustCompile(`sandbox|dropbox|auto`)

	apiUrls = map[string]string{
		"authorize-url": "https://www.dropbox.com/1/oauth2/authorize",

		"account/info":          "https://api.dropbox.com/1/account/info",
		"metadata":              "https://api.dropbox.com/1/metadata/<root>/<path>",
		"gets":                  "https://api-content.dropbox.com/1/files/<root>/<path>",
		"files_put":             "https://api-content.dropbox.com/1/files_put/<root>/<path>",
		"delta":                 "https://api.dropbox.com/1/delta",
		"revisions":             "https://api.dropbox.com/1/revisions/<root>/<path>",
		"restore":               "https://api.dropbox.com/1/restore/<root>/<path>",
		"search":                "https://api.dropbox.com/1/search/<root>/<path>",
		"shares":                "https://api.dropbox.com/1/shares/<root>/<path>",
		"media":                 "https://api.dropbox.com/1/media/<root>/<path>",
		"copy_ref":              "https://api.dropbox.com/1/copy_ref/<root>/<path>",
		"thumbnails":            "https://api-content.dropbox.com/1/thumbnails/<root>/<path>",
		"chunked_upload":        "https://api-content.dropbox.com/1/chunked_upload",
		"commit_chunked_upload": "https://api-content.dropbox.com/1/commit_chunked_upload/<root>/<path>",

		"fileops/copy":          "https://api.dropbox.com/1/fileops/copy",
		"fileops/create_folder": "https://api.dropbox.com/1/fileops/create_folder",
		"fileops/delete":        "https://api.dropbox.com/1/fileops/delete",
		"fileops/move":          "https://api.dropbox.com/1/fileops/move",
	}
)

type DropboxApi struct {
	Signer    RequestSinger
	Root      string // default root path
	Locale    string
	ErrorCode int
}

type ApiError struct {
	Code     int
	ErrorMsg string `json:"Error"`
}

func (err ApiError) Error() string {
	return err.ErrorMsg
}

type RequestSinger interface {
	Sign(*http.Request) *ApiError
}

type QuotaInfo struct {
	Shared int64
	Quota  int64
	Normal int64
}

type AccountInfo struct {
	Referral_link string
	Display_name  string
	Uid           int64
	Country       string
	Email         string
	Quota_info    QuotaInfo
}

type Content struct {
	Size         string
	Rev          string
	Thumb_exists bool
	Bytes        int
	Modified     string
	Client_mtime string
	Path         string
	Is_dir       bool
	Icon         string
	Root         string
	Mime_type    string
	Revision     int
}

type PathMetadata struct {
	Content
	Hash     string
	Contents []Content
}

func (api *DropboxApi) getUrl(name string) string {
	return apiUrls[name]
}

func (api *DropboxApi) getRootPathUrl(name, root, path string) string {
	apiUrl := api.getUrl(name)
	apiUrl = strings.Replace(apiUrl, "<root>", url.QueryEscape(root), 1)
	apiUrl = strings.Replace(apiUrl, "<path>", url.QueryEscape(path), 1)
	return apiUrl
}

func (api *DropboxApi) toApiError(err error) *ApiError {
	return &ApiError{Code: api.ErrorCode, ErrorMsg: err.Error()}
}

func (api *DropboxApi) getErrorMsg(body []byte, code int) *ApiError {
	msg := &ApiError{Code: code}
	json.Unmarshal(body, &msg)
	return msg
}

func AuthorizeUrl(appKey, redirectUrl string) {
	url := fmt.Sprintf("%s?response_type=token&client_id=%s&redirect_uri=%s", apiUrls["authorize-url"], appKey, redirectUrl)

	fmt.Printf("%s\n", url)
}

func (api *DropboxApi) doRequest(req *http.Request) (*http.Response, *ApiError) {
	if api.Signer == nil {
		return nil, &ApiError{Code: -1, ErrorMsg: "no Signer found ."}
	}

	err := api.Signer.Sign(req)
	if err != nil {
		return &http.Response{}, err
	}

	client := &http.Client{}

	resp, httperr := client.Do(req)
	if httperr != nil {
		err = api.toApiError(httperr)
	}
	return resp, err
}

func (api *DropboxApi) bytesToJson(bodybytes []byte, jsonObj interface{}) *ApiError {
	jsonerr := json.Unmarshal(bodybytes, &jsonObj)

	if jsonerr != nil {
		return api.toApiError(jsonerr)
	}

	return nil
}

func (api *DropboxApi) bodyToJson(resp *http.Response, jsonObj interface{}) *ApiError {
	bodybytes, ioerr := ioutil.ReadAll(resp.Body)

	if ioerr != nil {
		return api.toApiError(ioerr)
	}

	if resp.StatusCode == http.StatusOK {
		return api.bytesToJson(bodybytes, &jsonObj)
	} else {
		return api.getErrorMsg(bodybytes, resp.StatusCode)
	}
}

func (api *DropboxApi) doGet(url string) (*http.Response, *ApiError) {
	req, httperr := http.NewRequest("GET", url, nil)
	if httperr != nil {
		return nil, api.toApiError(httperr)
	}

	return api.doRequest(req)
}

func (api *DropboxApi) doPut(body io.Reader, url string) (*http.Response, *ApiError) {
	req, httperr := http.NewRequest("PUT", url, body)
	if httperr != nil {
		return nil, api.toApiError(httperr)
	}

	return api.doRequest(req)
}

func (api *DropboxApi) jsonRepsonse(method, url string, jsonObj interface{}) *ApiError {
	req, httperr := http.NewRequest(method, url, nil)
	if httperr != nil {
		return api.toApiError(httperr)
	}

	resp, err := api.doRequest(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return api.bodyToJson(resp, jsonObj)
}

func (api *DropboxApi) jsonReponseByGet(url string, jsonObj interface{}) *ApiError {
	return api.jsonRepsonse("GET", url, jsonObj)
}

func (api *DropboxApi) jsonReponseByPost(url string, jsonObj interface{}) *ApiError {
	return api.jsonRepsonse("POST", url, jsonObj)
}

func (api *DropboxApi) GetAccountInfo() (*AccountInfo, *ApiError) {
	url := api.getUrl("account/info")
	url = fmt.Sprintf("%s?locale=%s", url, api.Locale)

	var accountInfo = &AccountInfo{}

	err := api.jsonReponseByGet(url, accountInfo)

	return accountInfo, err
}

type FileEntry struct {
	Content
	DataByte []byte
}

func (api *DropboxApi) getFileEntry(apiurl string) (*FileEntry, *ApiError) {
	file := &FileEntry{}

	resp, err := api.doGet(apiurl)
	if err != nil {
		return file, err
	}

	defer resp.Body.Close()

	bytes, ioerr := ioutil.ReadAll(resp.Body)
	if ioerr != nil {
		return file, api.toApiError(ioerr)
	}
	file.DataByte = bytes

	metadata := resp.Header.Get("x-dropbox-metadata")
	err = api.bytesToJson([]byte(metadata), file)

	return file, err
}

func (api *DropboxApi) GetFile(path string) (*FileEntry, *ApiError) {
	return api.GetFile_(api.Root, path, "")
}

func (api *DropboxApi) GetFile_(root, path, rev string) (*FileEntry, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("gets", root, path)
	apiurl = fmt.Sprintf("%s?rev=%s", apiurl, rev)

	return api.getFileEntry(apiurl)
}

func (api *DropboxApi) Thumbnails(path string) (*FileEntry, *ApiError) {
	return api.Thumbnails_(api.Root, path, "jpeg", "s")
}

func (api *DropboxApi) Thumbnails_(root, path, format, size string) (*FileEntry, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("thumbnails", root, path)

	return api.getFileEntry(apiurl)
}

func (api *DropboxApi) GetFileMetadata(path string) (*PathMetadata, *ApiError) {
	return api.GetFileMetadata_(api.Root, path, 10000, "", true, false, "")
}

func (api *DropboxApi) GetFileMetadata_(root, path string, file_limit int, hash string,
	list, include_deleted bool, rev string) (*PathMetadata, *ApiError) {

	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("metadata", root, path)
	values := url.Values{}
	values.Add("file_limit", strconv.Itoa(file_limit))
	values.Add("hash", hash)
	values.Add("list", strconv.FormatBool(list))
	values.Add("include_deleted", strconv.FormatBool(include_deleted))
	values.Add("rev", rev)
	values.Add("locale", api.Locale)

	if v := values.Encode(); len(v) > 1 {
		apiurl = fmt.Sprintf("%s?%s", apiurl, v)
	}

	metadata := &PathMetadata{}
	err := api.jsonReponseByGet(apiurl, metadata)

	return metadata, err
}

func (api *DropboxApi) PutFileByName(localFilePath, path string) (*PathMetadata, *ApiError) {
	return api.PutFileByName_(localFilePath, api.Root, path)
}

func (api *DropboxApi) PutFileByName_(localFilePath, root, path string) (*PathMetadata, *ApiError) {
	file, ioerr := os.Open(localFilePath)
	if ioerr != nil {
		return &PathMetadata{}, api.toApiError(ioerr)
	}

	defer file.Close()

	return api.PutFileByReader(file, root, path)
}

func (api *DropboxApi) PutFileByReader(body io.Reader, root, path string) (*PathMetadata, *ApiError) {
	return api.PutFile(body, root, path, "", true)
}

func (api *DropboxApi) PutFile(body io.Reader, root, path, parent_rev string, overwrite bool) (*PathMetadata, *ApiError) {
	apiurl := api.getRootPathUrl("files_put", root, path)

	values := url.Values{}
	values.Add("overwrite", strconv.FormatBool(overwrite))
	values.Add("parent_rev", parent_rev)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &PathMetadata{}

	resp, err := api.doPut(body, apiurl)
	if err != nil {
		return metadata, err
	}

	defer resp.Body.Close()
	err = api.bodyToJson(resp, metadata)

	return metadata, err
}

type DeltaEntry struct {
	Path     string
	Metadata *PathMetadata
}

type DeltaResult struct {
	Entries []*DeltaEntry
	Reset   bool
	Cursor  string
	HasMore bool `json:"Has_more"`
}

type innerDeltaResult struct {
	Entries [][]interface{}
	Reset   bool
	Cursor  string
	HasMore bool `json:"Has_more"`
}

func (api *DropboxApi) Delta(cursor string) (*DeltaResult, *ApiError) {
	apiurl := api.getUrl("delta")

	values := url.Values{}
	values.Add("cursor", cursor)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	innerdelta := &innerDeltaResult{}
	err := api.jsonReponseByPost(apiurl, innerdelta)
	if err != nil {
		return nil, err
	}

	delta := &DeltaResult{Reset: innerdelta.Reset, Cursor: innerdelta.Cursor, HasMore: innerdelta.HasMore}
	entryCount := len(innerdelta.Entries)
	delta.Entries = make([]*DeltaEntry, entryCount)

	for i := 0; i < entryCount; i++ {
		v := innerdelta.Entries[i]

		str, _ := v[0].(string)
		entry := &DeltaEntry{Path: str}

		if v[1] != nil {
			entry.Metadata = api.convert2pathMetadata(v[1])
		}

		delta.Entries[i] = entry
	}

	return delta, nil
}

func (api *DropboxApi) convert2pathMetadata(val interface{}) *PathMetadata {
	metaMap := reflect.ValueOf(val)
	meta := &PathMetadata{}
	for _, key := range metaMap.MapKeys() {
		value := metaMap.MapIndex(key).Elem()
		fname := key.String()

		switch fname {
		case "revision":
			meta.Revision = int(value.Float())
		case "bytes":
			meta.Bytes = int(value.Float())
		case "is_dir":
			meta.Is_dir = value.Bool()
		case "thumb_exists":
			meta.Thumb_exists = value.Bool()
		case "modified":
			meta.Modified = value.String()
		case "rev":
			meta.Rev = value.String()
		case "path":
			meta.Path = value.String()
		case "icon":
			meta.Icon = value.String()
		case "root":
			meta.Root = value.String()
		case "size":
			meta.Size = value.String()
		}
	}

	return meta
}

func (api *DropboxApi) Revisions(path string) (*[]PathMetadata, *ApiError) {
	return api.Revisions_(api.Root, path, 10)
}

func (api *DropboxApi) Revisions_(root, path string, rev_limit int) (*[]PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("revisions", root, path)

	values := url.Values{}
	values.Add("rev_limit", strconv.Itoa(rev_limit))
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	res := &[]PathMetadata{}
	err := api.jsonReponseByPost(apiurl, res)
	return res, err
}

func (api *DropboxApi) Restore(path, rev string) (*PathMetadata, *ApiError) {
	return api.Restore_(api.Root, path, rev)
}

func (api *DropboxApi) Restore_(root, path, rev string) (*PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("restore", root, path)

	values := url.Values{}
	values.Add("rev", rev)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &PathMetadata{}
	err := api.jsonReponseByPost(apiurl, metadata)
	return metadata, err
}

func (api *DropboxApi) Search(path, query string) (*[]PathMetadata, *ApiError) {
	return api.Search_(api.Root, path, query, 1000, false)
}

func (api *DropboxApi) Search_(root, path, query string, file_limit int, include_deleted bool) (*[]PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("search", root, path)

	values := url.Values{}
	values.Add("query", query)
	values.Add("file_limit", strconv.Itoa(file_limit))
	values.Add("include_deleted", strconv.FormatBool(include_deleted))
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &[]PathMetadata{}
	err := api.jsonReponseByPost(apiurl, metadata)
	return metadata, err
}

func (api *DropboxApi) Shares(path string) (map[string]string, *ApiError) {
	return api.Shares_(api.Root, path, true)
}

func (api *DropboxApi) Shares_(root, path string, short_url bool) (map[string]string, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("shares", root, path)

	values := url.Values{}
	values.Add("short_url", strconv.FormatBool(short_url))
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := make(map[string]string)
	err := api.jsonReponseByPost(apiurl, &metadata)
	return metadata, err
}

func (api *DropboxApi) CopyRef(path string) (map[string]string, *ApiError) {
	return api.CopyRef_(api.Root, path)
}

func (api *DropboxApi) CopyRef_(root, path string) (map[string]string, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("copy_ref", root, path)

	metadata := make(map[string]string)
	err := api.jsonReponseByGet(apiurl, &metadata)
	return metadata, err
}

func (api *DropboxApi) Media(path string) (map[string]string, *ApiError) {
	return api.Media_(api.Root, path)
}

func (api *DropboxApi) Media_(root, path string) (map[string]string, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("media", root, path)

	values := url.Values{}
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := make(map[string]string)
	err := api.jsonReponseByGet(apiurl, &metadata)
	return metadata, err
}

type ChunkedUploadRes struct {
	Upload_id string
	Offset    int
	Expires   string
}

func (api *DropboxApi) UploadByChunked(localPath, path string, trunkSize, retryCount int) (*PathMetadata, *ApiError) {
	file, ioerr := os.Open(localPath)
	if ioerr != nil {
		return nil, api.toApiError(ioerr)
	}
	defer file.Close()

	return api.UploadReaderByChunked(file, path, trunkSize, retryCount)
}

func (api *DropboxApi) UploadReaderByChunked(file io.Reader, path string, trunkSize, retryCount int) (*PathMetadata, *ApiError) {
	buff := make([]byte, trunkSize)
	offset, uploadid := 0, ""

	for {
		n, ioerr := file.Read(buff)
		if ioerr == io.EOF {
			break
		}
		if ioerr != nil {
			return nil, api.toApiError(ioerr)
		}

		res, apiErr := api.retryUploadTrunk(buff[0:n], uploadid, offset, retryCount)
		if apiErr != nil {
			return nil, apiErr
		}
		offset, uploadid = res.Offset, res.Upload_id
	}

	return api.commitChunkedUpload(path, uploadid)
}

func (api *DropboxApi) retryUploadTrunk(trunk []byte, upload_id string, offset, retryCount int) (*ChunkedUploadRes, *ApiError) {
	var res *ChunkedUploadRes
	var err *ApiError
	for i := 1; i <= retryCount; i++ {
		res, err = api.chunkedUpload_(trunk, upload_id, offset)
		if err == nil {
			return res, nil
		} else if i == retryCount {
			return res, &ApiError{ErrorMsg: fmt.Sprintf("%s and cause too many upload retry times .", err.Error()), Code: api.ErrorCode}
		}
	}
	return res, err
}

func (api *DropboxApi) chunkedUpload_(trunk []byte, upload_id string, offset int) (*ChunkedUploadRes, *ApiError) {
	apiurl := api.getUrl("chunked_upload")

	values := url.Values{}
	if len(upload_id) > 0 {
		values.Add("upload_id", upload_id)
	}
	values.Add("offset", strconv.Itoa(offset))
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &ChunkedUploadRes{}
	resp, err := api.doPut(bytes.NewBuffer(trunk), apiurl)
	if err == nil {
		defer resp.Body.Close()
		err = api.bodyToJson(resp, metadata)
	}

	return metadata, err
}

func (api *DropboxApi) commitChunkedUpload(path, upload_id string) (*PathMetadata, *ApiError) {
	return api.CommitChunkedUpload_(api.Root, path, upload_id, "", true)
}

func (api *DropboxApi) CommitChunkedUpload_(root, path, upload_id, parent_rev string, overwrite bool) (*PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getRootPathUrl("commit_chunked_upload", root, path)

	values := url.Values{}
	values.Add("upload_id", upload_id)
	values.Add("overwrite", strconv.FormatBool(overwrite))
	values.Add("parent_rev", parent_rev)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}

func (api *DropboxApi) fileOpertaion(apiurl string) (*PathMetadata, *ApiError) {
	metadata := &PathMetadata{}
	err := api.jsonReponseByPost(apiurl, metadata)

	return metadata, err
}

func (api *DropboxApi) Copy(from_path, to_path string) (*PathMetadata, *ApiError) {
	return api.Copy_(api.Root, from_path, to_path, "")
}

func (api *DropboxApi) Copy_(root, from_path, to_path, from_copy_ref string) (*PathMetadata, *ApiError) {
	if hasNil([]string{root, to_path}) {
		return nil, &ApiError{Code: -1, ErrorMsg: "root, to_path are all required ."}
	}
	if !hasNotNil([]string{from_path, from_copy_ref}) {
		return nil, &ApiError{Code: -1, ErrorMsg: "from_path, from_copy_ref must have one non-nil value ."}
	}

	if err := checkRoot(root); err != nil {
		return nil, err
	}

	apiurl := api.getUrl("fileops/copy")

	values := url.Values{}
	values.Add("root", root)
	values.Add("from_path", from_path)
	values.Add("to_path", to_path)
	values.Add("from_copy_ref", from_copy_ref)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}

func (api *DropboxApi) CreateFolder(path string) (*PathMetadata, *ApiError) {
	return api.CreateFolder_(api.Root, path)
}

func (api *DropboxApi) CreateFolder_(root, path string) (*PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getUrl("fileops/create_folder")

	values := url.Values{}
	values.Add("root", root)
	values.Add("path", path)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}

func (api *DropboxApi) Delete(path string) (*PathMetadata, *ApiError) {
	return api.Delete_(api.Root, path)
}

func (api *DropboxApi) Delete_(root, path string) (*PathMetadata, *ApiError) {
	if err := checkRootAndPath(root, path); err != nil {
		return nil, err
	}

	apiurl := api.getUrl("fileops/delete")

	values := url.Values{}
	values.Add("root", root)
	values.Add("path", path)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}

func (api *DropboxApi) Move(from_path, to_path string) (*PathMetadata, *ApiError) {
	return api.Move_(api.Root, from_path, to_path)
}

func (api *DropboxApi) Move_(root, from_path, to_path string) (*PathMetadata, *ApiError) {
	if hasNil([]string{root, from_path, to_path}) {
		return nil, &ApiError{Code: -1, ErrorMsg: "root, from_path, to_path are all required ."}
	}

	if err := checkRoot(root); err != nil {
		return nil, err
	}

	apiurl := api.getUrl("fileops/move")

	values := url.Values{}
	values.Add("root", root)
	values.Add("from_path", from_path)
	values.Add("to_path", to_path)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}

func checkRootAndPath(root, path string) *ApiError {
	if hasNil([]string{root, path}) {
		return &ApiError{Code: -1, ErrorMsg: "root, path are all required ."}
	}

	return checkRoot(root)
}

func checkRoot(root string) *ApiError {
	if rootRegexp.MatchString(root) {
		return nil
	}

	return &ApiError{Code: -1, ErrorMsg: `root must be "dropbox" or "sandbox" or "auto" .`}
}

func exists(strs []string, judge func(string) bool) bool {
	for _, str := range strs {
		if judge(str) {
			return true
		}
	}
	return false
}

func hasNil(strs []string) bool {
	judge := func(str string) bool {
		return len(str) == 0
	}

	return exists(strs, judge)
}

func hasNotNil(strs []string) bool {
	judge := func(str string) bool {
		return len(str) > 0
	}

	return exists(strs, judge)
}
