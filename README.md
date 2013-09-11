godropbox
=========

dropbox sdk implemented in Go .

###  get accessToken
construct a url like below:   
https://www.dropbox.com/1/oauth2/authorize?response_type=token&client_id=you-appKey&redirect_uri=you-redirect-url

then open in browser, after redirected, the address in browser may look like below :   
http://127.0.0.1/authorized#access_token=zC_ZXIYlO8QAAAAAAAAAARGiMdIq6QDFUO46EreBouy6oesz-0XsK9qYJbrqEIIp&token_type=bearer&uid=158130000

now, you get accessToken !

###  Example
you can get more example in file dropbox_test.go .

~~~Go
 oauth2 := &dropbox.OAuth2{AccessToken: "you ouath2 access_token"}

 dropboxApi := &dropbox.DropboxApi{Signer: oauth2, Root: "dropbox", Locale: "CN"}

 accountInfo, err := dropboxApi.GetAccountInfo()
 if err != nil {
	 fmt.Printf("error msg: %s\n", err)
 } else {
	 fmt.Printf("account info: %v\n", accountInfo)
 }

 metadata, err := dropboxApi.GetFileMetadata("/")
 if err != nil {
	 fmt.Printf("error msg: %s\n", err)
 } else {
	 fmt.Printf("metadata: %v\n", metadata)
 }
 
 put, err := dropboxApi.PutFileByName("main.go", "dropbox", "/main.go")
 if err != nil {
     fmt.Printf("error msg: %s\n", err)
 } else {
     fmt.Printf("put: %v\n", put)
 }

 copyRef, err := dropboxApi.CopyRef("/main.go")
 if err != nil {
     Fmt.Printf("error msg: %s\n", err)
 } else {
     fmt.Printf("copyRef : %v\n", copyRef)
 }

 revisions, err := dropboxApi.Revisions("/main.go")
 if err != nil {
     fmt.Printf("error msg: %s\n", err)
 } else {
     fmt.Printf("revisions : %v\n", revisions)
 }

 shares, err := dropboxApi.Shares("/main.go")
 if err != nil {
     fmt.Printf("error msg: %s\n", err)
 } else {
     fmt.Printf("shares : %v\n", shares)
 }
 
  media, err := dropboxApi.Media("/main.go")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("media : %v\n", media)
  }

  thumbnails, err := dropboxApi.Thumbnails("/IMG_20130613_121901.jpg")
  if err != nil {
      fmt.Printf("get thumbnails error msg: %s\n", err)
  } else {
      ioerr := ioutil.WriteFile("IMG_20130613_121901.jpg", thumbnails.DataByte, 666)
      if ioerr == nil {
          fmt.Printf("write image ok .\n")
      } else {
          fmt.Printf("write image error : %v\n", ioerr)
      }
  }

  copym, err := dropboxApi.Copy("/testcopy.txt", "/abctest/testcopy.txt")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("copym : %v\n", copym)
  }

  copym, err := dropboxApi.Copy("/testcopy.txt", "/abctest/testcopy.txt")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("copym : %v\n", copym)
  }

  move, err := dropboxApi.Move("/abctest/testcopy.txt", "/testcopy-moved.txt")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("move : %v\n", move)
  }

  createFolder, err := dropboxApi.CreateFolder("createFolder")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("createFolder: %v\n", createFolder)
  }

  deleted, err := dropboxApi.Delete("createFolder")
  if err != nil {
      fmt.Printf("error msg: %s\n", err)
  } else {
      fmt.Printf("deleted: %v\n", deleted)
  }
~~~
