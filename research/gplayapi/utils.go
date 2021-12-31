package gplayapi

import (
   "encoding/json"
   "errors"
   "fmt"
   "google.golang.org/protobuf/proto"
   "gplayapi/gpproto"
   "io"
   "net/http"
   "net/http/httputil"
   "os"
   "strings"
)

func doReq(r *http.Request) ([]byte, int, error) {
   buf, err := httputil.DumpRequest(r, true)
   if err != nil {
      return nil, 0, err
   }
   fmt.Printf("%q\n\n", buf)
   res, err := httpClient.Do(r)
   if err != nil {
      return nil, 0, err
   }
   defer res.Body.Close()
   b, err := io.ReadAll(res.Body)
   return b, res.StatusCode, err
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrStr(str string) *string {
	return &str
}

func ptrInt32(i int32) *int32 {
	return &i
}

func parseResponse(res string) map[string]string {
	ret := map[string]string{}
	for _, ln := range strings.Split(res, "\n") {
		keyVal := strings.SplitN(ln, "=", 2)
		if len(keyVal) >= 2 {
			ret[keyVal[0]] = keyVal[1]
		}
	}
	return ret
}

func (client *GooglePlayClient) _doAuthedReq(r *http.Request) (_ *gpproto.Payload, err error) {
	client.setDefaultHeaders(r)
	b, status, err := doReq(r)
	if err != nil {
		return
	}
	if status == 401 {
		return nil, GPTokenExpired
	}
	resp := &gpproto.ResponseWrapper{}
	err = proto.Unmarshal(b, resp)
	if err != nil {
		return
	}
	return resp.Payload, nil
}

func (client *GooglePlayClient) doAuthedReq(r *http.Request) (res *gpproto.Payload, err error) {
	res, err = client._doAuthedReq(r)
	if err == GPTokenExpired {
		err = client.RegenerateGPToken()
		if err != nil {
			return
		}
		if client.SessionFile != "" {
			client.SaveSession(client.SessionFile)
		}
		res, err = client._doAuthedReq(r)
	}
	return
}

func (client *GooglePlayClient) RegenerateGPToken() (err error) {
	client.AuthData.AuthToken, err = client.GenerateGPToken()
	return
}

const (
	ImageTypeAppScreenshot = iota + 1
	ImageTypePlayStorePageBackground
	ImageTypeYoutubeVideoLink
	ImageTypeAppIcon
	ImageTypeCategoryIcon
	ImageTypeYoutubeVideoThumbnail = 13

	UrlBase               = "https://android.clients.google.com"
	UrlFdfe               = UrlBase + "/fdfe"
	UrlAuth               = UrlBase + "/auth"
	UrlCheckIn            = UrlBase + "/checkin"
	UrlDetails            = UrlFdfe + "/details"
	UrlDelivery           = UrlFdfe + "/delivery"
	UrlPurchase           = UrlFdfe + "/purchase"
	UrlToc                = UrlFdfe + "/toc"
	UrlTosAccept          = UrlFdfe + "/acceptTos"
	UrlUploadDeviceConfig = UrlFdfe + "/uploadDeviceConfig"
)

type GooglePlayClient struct {
	AuthData   *AuthData
	DeviceInfo *DeviceInfo

	// SessionFile if SessionFile is set then session will be saved to it after modification
	SessionFile string
}

var (
	GPTokenExpired = errors.New("unauthorized, gp token expired")
	httpClient = &http.Client{}
)

func NewClient(email, aasToken string) (*GooglePlayClient, error) {
	return NewClientWithDeviceInfo(email, aasToken, Pixel3a)
}

func NewClientWithDeviceInfo(email, aasToken string, deviceInfo *DeviceInfo) (client *GooglePlayClient, err error) {
   authData := &AuthData{
   Email:    email,
   AASToken: aasToken,
   Locale:   "en_GB",
   }
   client = &GooglePlayClient{AuthData: authData, DeviceInfo: deviceInfo}
   _, err = client.GenerateGsfID()
   if err != nil {
   return
   }
   deviceConfigRes, err := client.uploadDeviceConfig()
   if err != nil {
   return
   }
   authData.DeviceConfigToken = deviceConfigRes.GetUploadDeviceConfigToken()
   token, err := client.GenerateGPToken()
   if err != nil {
   return
   }
   authData.AuthToken = token
   return
}

func (client *GooglePlayClient) SaveSession(file string) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(client.AuthData)
}