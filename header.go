package googleplay

import (
	"github.com/TravelSir/format/protobuf"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Header struct {
	http.Header
}

func (h Header) Delivery(app string, ver uint64) (*Delivery, error) {
	req, err := http.NewRequest(
		"GET", "https://play-fe.googleapis.com/fdfe/delivery", nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header = h.Header
	req.URL.RawQuery = url.Values{
		"doc": {app},
		"vc":  {strconv.FormatUint(ver, 10)},
	}.Encode()
	LogLevel.Dump(req)
	res, err := new(http.Transport).RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	responseWrapper, err := protobuf.Decode(res.Body)
	if err != nil {
		return nil, err
	}
	// .payload.deliveryResponse.status
	status, err := responseWrapper.Get(1).Get(21).GetVarint(1)
	if err != nil {
		return nil, err
	}
	switch status {
	case 2:
		return nil, errorString("Geo-blocking")
	case 3:
		return nil, errorString("Purchase required")
	case 5:
		return nil, errorString("Invalid version")
	}
	// .payload.deliveryResponse.appDeliveryData
	appData := responseWrapper.Get(1).Get(21).Get(2)
	var del Delivery
	// downloadUrl
	del.DownloadURL, err = appData.GetString(3)
	if err != nil {
		return nil, err
	}
	// .splitDeliveryData
	for _, data := range appData.GetMessages(15) {
		var split SplitDeliveryData
		// .id
		split.ID, err = data.GetString(1)
		if err != nil {
			return nil, err
		}
		// .downloadUrl
		split.DownloadURL, err = data.GetString(5)
		if err != nil {
			return nil, err
		}
		del.SplitDeliveryData = append(del.SplitDeliveryData, split)
	}
	return &del, nil
}

func (h Header) Details(app string) (*Details, error) {
	req, err := http.NewRequest(
		"GET", "https://android.clients.google.com/fdfe/details", nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header = h.Header
	req.URL.RawQuery = "doc=" + url.QueryEscape(app)
	LogLevel.Dump(req)
	res, err := new(http.Transport).RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, errorString(res.Status)
	}
	responseWrapper, err := protobuf.Decode(res.Body)
	if err != nil {
		return nil, err
	}
	// .payload.detailsResponse.docV2
	docV2 := responseWrapper.Get(1).Get(2).Get(4)
	var det Details
	// .creator
	det.Creator, err = docV2.GetString(6)
	if err != nil {
		return nil, err
	}
	// .offer.currencyCode
	det.CurrencyCode, err = docV2.Get(8).GetString(2)
	if err != nil {
		return nil, err
	}
	// .details.appDetails.file
	files := docV2.Get(13).Get(1).GetMessages(17)
	det.Files = len(files)
	// .offer.micros
	det.Micros, err = docV2.Get(8).GetVarint(1)
	if err != nil {
		return nil, err
	}
	// I dont know the name of field 70
	// .details.appDetails
	det.NumDownloads, err = docV2.Get(13).Get(1).GetVarint(70)
	if err != nil {
		return nil, err
	}
	// The shorter path 13,1,9 returns wrong size for some packages:
	// com.riotgames.league.wildriftvn
	// .details.appDetails.installDetails.size
	det.Size, err = docV2.Get(13).Get(1).Get(34).GetVarint(2)
	if err != nil {
		return nil, err
	}
	// .title
	det.Title, err = docV2.GetString(5)
	if err != nil {
		return nil, err
	}
	// .details.appDetails.uploadDate
	det.UploadDate, err = docV2.Get(13).Get(1).GetString(16)
	if err != nil {
		return nil, err
	}
	// .details.appDetails.versionCode
	det.VersionCode, err = docV2.Get(13).Get(1).GetVarint(3)
	if err != nil {
		return nil, err
	}
	// .details.appDetails.versionString
	det.VersionString, err = docV2.Get(13).Get(1).GetString(4)
	if err != nil {
		return nil, err
	}
	return &det, nil
}

// Purchase app. Only needs to be done once per Google account.
func (h Header) Purchase(app string) error {
	query := "doc=" + url.QueryEscape(app)
	req, err := http.NewRequest(
		"POST", "https://android.clients.google.com/fdfe/purchase",
		strings.NewReader(query),
	)
	if err != nil {
		return err
	}
	h.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header = h.Header
	LogLevel.Dump(req)
	res, err := new(http.Transport).RoundTrip(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errorString(res.Status)
	}
	return nil
}
