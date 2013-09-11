package dropbox

import (
	"fmt"
	"net/http"
)

type OAuth2 struct {
	AccessToken string
	TokenType   string
	Uid         string
}

func (oauth *OAuth2) Sign(req *http.Request) *ApiError {
	authHeader := fmt.Sprintf("Bearer %s", oauth.AccessToken)
	req.Header.Add("Authorization", authHeader)
	return nil
}
