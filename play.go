package googleplay

import (
	"bufio"
	"fmt"
	"github.com/TravelSir/format"
	"github.com/TravelSir/format/crypto"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Sleep = 4 * time.Second

var LogLevel format.LogLevel

func parseQuery(query io.Reader) url.Values {
	vals := make(url.Values)
	buf := bufio.NewScanner(query)
	for buf.Scan() {
		key, val, ok := strings.Cut(buf.Text(), "=")
		if ok {
			vals.Add(key, val)
		}
	}
	return vals
}

type Delivery struct {
	DownloadURL       String
	SplitDeliveryData []SplitDeliveryData
}

func (d Delivery) Data() []SplitDeliveryData {
	datas := d.SplitDeliveryData
	data := SplitDeliveryData{DownloadURL: d.DownloadURL}
	return append(datas, data)
}

type Details struct {
	Title         String
	Creator       String
	UploadDate    String
	VersionString String
	VersionCode   Varint
	NumDownloads  Varint
	Size          Varint
	Files         int
	Micros        Varint
	CurrencyCode  String
}

func (d Details) Format(f fmt.State, verb rune) {
	fmt.Fprintln(f, "Title:", d.Title)
	fmt.Fprintln(f, "Creator:", d.Creator)
	fmt.Fprintln(f, "UploadDate:", d.UploadDate)
	fmt.Fprintln(f, "VersionString:", d.VersionString)
	fmt.Fprintln(f, "VersionCode:", d.VersionCode)
	fmt.Fprintln(f, "NumDownloads:", format.LabelNumber(d.NumDownloads))
	fmt.Fprintln(f, "Size:", format.LabelSize(d.Size))
	fmt.Fprintln(f, "Files:", d.Files)
	fmt.Fprint(f, "Offer: ", d.Micros, " ", d.CurrencyCode)
}

type SplitDeliveryData struct {
	ID          String
	DownloadURL String
}

func (s SplitDeliveryData) Name(app string, ver uint64) string {
	if s.ID != "" {
		return fmt.Sprint(app, "-", s.ID, "-", ver, ".apk")
	}
	return fmt.Sprint(app, "-", ver, ".apk")
}

type Token struct {
	Services string
	Token    string
}

// You can also use host "android.clients.google.com", but it also uses
// TLS fingerprinting.
func NewToken(email, password string, proxy string) (*Token, error) {
	body := url.Values{
		"Email":              {email},
		"Passwd":             {password},
		"client_sig":         {""},
		"droidguard_results": {""},
	}.Encode()
	req, err := http.NewRequest(
		"POST", "https://android.googleapis.com/auth", strings.NewReader(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	hello, err := crypto.ParseJA3(crypto.AndroidAPI26)
	if err != nil {
		return nil, err
	}
	LogLevel.Dump(req)
	res, err := crypto.Transport(hello, proxy).RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errorString(res.Status)
	}
	val := parseQuery(res.Body)
	var tok Token
	tok.Services = val.Get("services")
	tok.Token = val.Get("Token")
	return &tok, nil
}

func OpenToken(elem ...string) (*Token, error) {
	return format.Open[Token](elem...)
}

func (t Token) Create(elem ...string) error {
	return format.Create(t, elem...)
}

func (t Token) Header(dev *Device) (*Header, error) {
	return t.headerVersion(dev, 9999_9999)
}

func (t Token) SingleAPK(dev *Device) (*Header, error) {
	return t.headerVersion(dev, 8091_9999)
}

func (t Token) headerVersion(dev *Device, version int64) (*Header, error) {
	val := url.Values{
		"Token":   {t.Token},
		"service": {"oauth2:https://www.googleapis.com/auth/googleplay"},
	}.Encode()
	req, err := http.NewRequest(
		"POST", "https://android.googleapis.com/auth", strings.NewReader(val),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	LogLevel.Dump(req)
	res, err := new(http.Transport).RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errorString(res.Status)
	}
	var head Header
	head.Header = make(http.Header)
	// Authorization
	head.Set(
		"Authorization",
		"Bearer "+parseQuery(res.Body).Get("Auth"),
	)
	// User-Agent
	head.Set(
		"User-Agent",
		fmt.Sprintf("Android-Finsky (sdk=9,versionCode=%v", version),
	)
	// X-DFE-Device-ID
	head.Set(
		"X-DFE-Device-ID",
		fmt.Sprintf("%x", dev.AndroidID),
	)
	return &head, nil
}

type errorString string

func (e errorString) Error() string {
	return string(e)
}
