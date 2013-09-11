package dropbox

import (
	"testing"
)

var (
	accessToken = "E6KBMoN1FSkAAAAAAAAAAV1m7sZCWsVVUzvrwwUMqhCTz0pB6_SJyiYh0Q9IepsD"
)

func getApi() *DropboxApi {
	oauth2 := &OAuth2{AccessToken: accessToken}
	dropboxApi := &DropboxApi{Signer: oauth2, Root: "sandbox", Locale: "CN"}
	return dropboxApi
}

func TestGetAccountInfo(t *testing.T) {
	dropboxApi := getApi()

	accountInfo, err := dropboxApi.GetAccountInfo()
	if err != nil {
		t.Logf("account msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("account info: %v\n", accountInfo)
	}
}

func TestGetFileMetadata(t *testing.T) {
	dropboxApi := getApi()

	metadata, err := dropboxApi.GetFileMetadata("/")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("metadata: %v\n", metadata)
	}
}

func TestPutFileByName(t *testing.T) {
	dropboxApi := getApi()

	put, err := dropboxApi.PutFileByName("dropbox.go", "dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("put: %v\n", put)
	}
}

func TestCopyRef(t *testing.T) {
	dropboxApi := getApi()

	copyRef, err := dropboxApi.CopyRef("dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("copyRef : %v\n", copyRef)
	}
}

func TestRevisions(t *testing.T) {
	dropboxApi := getApi()

	revisions, err := dropboxApi.Revisions("/dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("revisions : %v\n", revisions)
	}
}

func TestShares(t *testing.T) {
	dropboxApi := getApi()

	shares, err := dropboxApi.Shares("/dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("shares : %v\n", shares)
	}
}

func TestMedia(t *testing.T) {
	dropboxApi := getApi()

	media, err := dropboxApi.Media("/dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("media : %v\n", media)
	}
}

func TestThumbnails(t *testing.T) {
	dropboxApi := getApi()

	_, err := dropboxApi.Thumbnails("/IMG_20130613_121901.jpg")
	if err != nil {
		t.Logf("get thumbnails error msg: %s\n", err)
		t.Fail()
	}
}

func TestCopy(t *testing.T) {
	dropboxApi := getApi()

	dropboxApi.Delete("/copy-dropbox.go")
	copym, err := dropboxApi.Copy("/dropbox.go", "/copy-dropbox.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("copym : %v\n", copym)
	}
}

func TestMove(t *testing.T) {
	dropboxApi := getApi()

	dropboxApi.Delete("/dropbox-moved.go")
	move, err := dropboxApi.Move("dropbox.go", "/dropbox-moved.go")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("move : %v\n", move)
	}
}

func TestCreateFolder(t *testing.T) {
	dropboxApi := getApi()

	createFolder, err := dropboxApi.CreateFolder("createFolder")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("createFolder: %v\n", createFolder)
	}
}

func TestDelete(t *testing.T) {
	dropboxApi := getApi()

	deleted, err := dropboxApi.Delete("createFolder")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("deleted: %v\n", deleted)
	}
}

func TestDelta(t *testing.T) {
	dropboxApi := getApi()

	delta, err := dropboxApi.Delta("")
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		for _, v := range delta.Entries {
			t.Logf("path:%s, metadata: %v\n", v.Path, v.Metadata)
		}
		t.Logf("new_cursor:%s\n", delta.Cursor)
	}
}

func TestUploadByChunked(t *testing.T) {
	//dropbox.Authorize(appKey, "https://coderbee.net/oauth2/authorized")
	dropboxApi := getApi()

	trunked, err := dropboxApi.UploadByChunked("dropbox.go", "dropbox.go-byTrunked", 10240, 2)
	if err != nil {
		t.Logf("error msg: %s\n", err)
		t.Fail()
	} else {
		t.Logf("trunked: %v\n", trunked)
	}
}
