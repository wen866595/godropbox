package oauth2

import (
	"../../dropbox"
	"fmt"
	"net/http"
)

type OAuth2 struct {
	AccessToken string
	TokenType   string
	Uid         string
}

func (oauth *OAuth2) Sign(req *http.Request) *dropbox.ApiError {
	authHeader := fmt.Sprintf("Bearer %s", oauth.AccessToken)
	req.Header.Add("Authorization", authHeader)
	return nil
}
