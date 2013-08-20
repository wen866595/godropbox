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
	"strconv"
	"strings"
)

var (
	ApiUrls = map[string]string{
		"authorize-url":       "https://www.dropbox.com/1/oauth2/authorize",
		"authorized-redirect": "https://localhost/oauth2/authorized",

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
	Shared int
	Quota  int
	Normal int
}

type AccountInfo struct {
	Referral_link string
	Display_name  string
	Uid           int
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

type FileMetadata struct {
	Content
}

type PathMetadata struct {
	Content
	Hash     string
	Contents []Content
}

func (api *DropboxApi) GetUrl(name string) string {
	return ApiUrls[name]
}

func (api *DropboxApi) GetRootPathUrl(name, root, path string) string {
	apiUrl := api.GetUrl(name)
	apiUrl = strings.Replace(apiUrl, "<root>", url.QueryEscape(root), 1)
	apiUrl = strings.Replace(apiUrl, "<path>", url.QueryEscape(path), 1)
	return apiUrl
}

func (api *DropboxApi) ToApiError(err error) *ApiError {
	return &ApiError{Code: api.ErrorCode, ErrorMsg: err.Error()}
}

func (api *DropboxApi) GetErrorMsg(body []byte, code int) *ApiError {
	msg := &ApiError{Code: code}
	json.Unmarshal(body, &msg)
	return msg
}

func (api *DropboxApi) Authorize(appKey string) {
	url := fmt.Sprintf("%s?response_type=token&client_id=%s&redirect_uri=%s", api.GetUrl("authorize-url"), appKey, api.GetUrl("authorized-redirect"))
	// https://coderbee.net/oauth2/authorized#access_token=O_YKHHDEy3kAAAAAAAAAAVZZU1K72vMSH9U8LcgK83_jjm2R95bWelhC7qpbEbwX&token_type=bearer&uid=158135984

	fmt.Printf("%s\n", url)
}

func (api *DropboxApi) DoRequest(req *http.Request) (*http.Response, *ApiError) {
	err := api.Signer.Sign(req)
	if err != nil {
		return &http.Response{}, err
	}

	client := &http.Client{}

	resp, httperr := client.Do(req)
	if httperr != nil {
		err = api.ToApiError(httperr)
	}
	return resp, err
}

func (api *DropboxApi) bytesToJson(bodybytes []byte, jsonObj interface{}) *ApiError {
	jsonerr := json.Unmarshal(bodybytes, &jsonObj)

	if jsonerr != nil {
		return api.ToApiError(jsonerr)
	}

	return nil
}

func (api *DropboxApi) bodyToJson(resp *http.Response, jsonObj interface{}) *ApiError {
	bodybytes, ioerr := ioutil.ReadAll(resp.Body)

	if ioerr != nil {
		return api.ToApiError(ioerr)
	}

	if resp.StatusCode == http.StatusOK {
		return api.bytesToJson(bodybytes, &jsonObj)
	} else {
		return api.GetErrorMsg(bodybytes, resp.StatusCode)
	}
}

func (api *DropboxApi) DoGet(url string) (*http.Response, *ApiError) {
	req, httperr := http.NewRequest("GET", url, nil)
	if httperr != nil {
		return nil, api.ToApiError(httperr)
	}

	return api.DoRequest(req)
}

func (api *DropboxApi) DoPut(body io.Reader, url string) (*http.Response, *ApiError) {
	req, httperr := http.NewRequest("PUT", url, body)
	if httperr != nil {
		return nil, api.ToApiError(httperr)
	}

	return api.DoRequest(req)
}

func (api *DropboxApi) jsonRepsonse(method, url string, jsonObj interface{}) *ApiError {
	req, httperr := http.NewRequest(method, url, nil)
	if httperr != nil {
		return api.ToApiError(httperr)
	}

	resp, err := api.DoRequest(req)
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
	url := api.GetUrl("account/info")
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

	resp, err := api.DoGet(apiurl)
	if err != nil {
		return file, err
	}

	defer resp.Body.Close()

	bytes, ioerr := ioutil.ReadAll(resp.Body)
	if ioerr != nil {
		return file, api.ToApiError(ioerr)
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
	apiurl := api.GetRootPathUrl("gets", root, path)
	apiurl = fmt.Sprintf("%s?rev=%s", apiurl, rev)

	return api.getFileEntry(apiurl)
}

func (api *DropboxApi) Thumbnails(path string) (*FileEntry, *ApiError) {
	return api.Thumbnails_(api.Root, path, "jpeg", "s")
}

func (api *DropboxApi) Thumbnails_(root, path, format, size string) (*FileEntry, *ApiError) {
	apiurl := api.GetRootPathUrl("thumbnails", root, path)

	return api.getFileEntry(apiurl)
}

func (api *DropboxApi) GetFileMetadata(path string) (*PathMetadata, *ApiError) {
	return api.GetFileMetadata_(api.Root, path, 10000, "", true, false, "")
}

func (api *DropboxApi) GetFileMetadata_(root, path string, file_limit int, hash string,
	list, include_deleted bool, rev string) (*PathMetadata, *ApiError) {

	apiurl := api.GetRootPathUrl("metadata", root, path)
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

func (api *DropboxApi) PutFileByName(localFilePath, root, path string) (*PathMetadata, *ApiError) {
	file, ioerr := os.Open(localFilePath)
	if ioerr != nil {
		return &PathMetadata{}, api.ToApiError(ioerr)
	}

	defer file.Close()

	return api.PutFileByReader(file, root, path)
}

func (api *DropboxApi) PutFileByReader(body io.Reader, root, path string) (*PathMetadata, *ApiError) {
	return api.PutFile(body, root, path, "", true)
}

func (api *DropboxApi) PutFile(body io.Reader, root, path, parent_rev string, overwrite bool) (*PathMetadata, *ApiError) {
	apiurl := api.GetRootPathUrl("files_put", root, path)

	values := url.Values{}
	values.Add("overwrite", strconv.FormatBool(overwrite))
	values.Add("parent_rev", parent_rev)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &PathMetadata{}

	resp, err := api.DoPut(body, apiurl)
	if err != nil {
		return metadata, err
	}

	defer resp.Body.Close()
	err = api.bodyToJson(resp, metadata)

	return metadata, err
}

type DeltaEntry struct {
	Path     string
	Metadata PathMetadata
}

type DeltaResult struct {
	Entries [][]DeltaEntry
	Reset   bool
	Cursor  string
	HasMore bool `json:"Has_more"`
}

func (api *DropboxApi) Delta(cursor string) (*DeltaResult, *ApiError) {
	apiurl := api.GetUrl("delta")

	values := url.Values{}
	values.Add("cursor", cursor)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	delta := &DeltaResult{}
	err := api.jsonReponseByPost(apiurl, delta)

	return delta, err
}

func (api *DropboxApi) Revisions(path string) (*[]PathMetadata, *ApiError) {
	return api.Revisions_(api.Root, path, 10)
}

func (api *DropboxApi) Revisions_(root, path string, rev_limit int) (*[]PathMetadata, *ApiError) {
	apiurl := api.GetRootPathUrl("revisions", root, path)

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
	apiurl := api.GetRootPathUrl("restore", root, path)

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
	apiurl := api.GetRootPathUrl("search", root, path)

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
	apiurl := api.GetRootPathUrl("shares", root, path)
	fmt.Printf("apiurl:%s\n", apiurl)

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
	apiurl := api.GetRootPathUrl("copy_ref", root, path)
	fmt.Printf("apiurl:%s\n", apiurl)

	metadata := make(map[string]string)
	err := api.jsonReponseByGet(apiurl, &metadata)
	return metadata, err
}

func (api *DropboxApi) Media(path string) (map[string]string, *ApiError) {
	return api.Media_(api.Root, path)
}

func (api *DropboxApi) Media_(root, path string) (map[string]string, *ApiError) {
	apiurl := api.GetRootPathUrl("media", root, path)

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

func (api *DropboxApi) UploadByChunked(localPath string, trunkSize, totalRetryCount int) (*ChunkedUploadRes, *ApiError) {
    file, ioerr := os.Open(localPath)
    if ioerr != nil {
        return nil, api.ToApiError(ioerr)
    }
    defer file.Close()

    return api.UploadReaderByChunked(file, trunkSize, totalRetryCount)
}

func (api *DropboxApi) UploadReaderByChunked(file io.Reader, trunkSize, totalRetryCount int) (*ChunkedUploadRes, *ApiError) {
    
    return nil, nil
}

func (api *DropboxApi) ChunkedUpload_(trunk []byte, upload_id string, offset int) (*ChunkedUploadRes, *ApiError) {
	apiurl := api.GetUrl("chunked_upload")

	values := url.Values{}
	values.Add("upload_id", upload_id)
	values.Add("offset", strconv.Itoa(offset))
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	metadata := &ChunkedUploadRes{}
	resp, err := api.DoPut(bytes.NewBuffer(trunk), apiurl)
	if err == nil {
		defer resp.Body.Close()
		err = api.bodyToJson(resp, metadata)
	}

	return metadata, err
}

func (api *DropboxApi) CommitChunkedUpload(path, upload_id string) (*PathMetadata, *ApiError) {
	return api.CommitChunkedUpload_(api.Root, path, upload_id, "", true)
}

func (api *DropboxApi) CommitChunkedUpload_(root, path, upload_id, parent_rev string, overwrite bool) (*PathMetadata, *ApiError) {
	apiurl := api.GetRootPathUrl("commit_chunked_upload", root, path)

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
	apiurl := api.GetUrl("fileops/copy")

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
	apiurl := api.GetUrl("fileops/create_folder")

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
	apiurl := api.GetUrl("fileops/delete")

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
	apiurl := api.GetUrl("fileops/move")

	values := url.Values{}
	values.Add("root", root)
	values.Add("from_path", from_path)
	values.Add("to_path", to_path)
	values.Add("locale", api.Locale)
	apiurl = fmt.Sprintf("%s?%s", apiurl, values.Encode())

	return api.fileOpertaion(apiurl)
}
