package main

import (
	"fmt"
	"github.com/TravelSir/format"
	gp "github.com/TravelSir/googleplay"
	"io"
	"net/http"
	"os"
	"time"
)

func doToken(email, password string, s5addr, s5user, s5pwd string) error {
	tok, err := gp.NewToken(email, password, s5addr, s5user, s5pwd)
	if err != nil {
		return err
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	return tok.Create(cache, "googleplay/token.json")
}

func doDelivery(head *gp.Header, app string, ver uint64) error {
	del, err := head.Delivery(app, ver)
	if err != nil {
		return err
	}
	for _, data := range del.Data() {
		fmt.Println("GET", data.DownloadURL)
		res, err := http.Get(string(data.DownloadURL))
		if err != nil {
			return err
		}
		file, err := os.Create(data.Name(app, ver))
		if err != nil {
			return err
		}
		pro := format.ProgressBytes(file, res.ContentLength)
		if _, err := io.Copy(pro, res.Body); err != nil {
			return err
		}
		if err := res.Body.Close(); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}

type native struct {
	path     string
	platform gp.String
}

func newNative(armeabi, arm64 bool) native {
	if armeabi {
		return native{"googleplay/armeabi.json", gp.Armeabi}
	}
	if arm64 {
		return native{"googleplay/arm64.json", gp.Arm64}
	}
	return native{"googleplay/x86.json", gp.X86}
}

func (n native) device() error {
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	device, err := gp.Phone.Checkin(n.platform)
	if err != nil {
		return err
	}
	fmt.Printf("Sleeping %v for server to process\n", gp.Sleep)
	time.Sleep(gp.Sleep)
	return device.Create(cache, n.path)
}

func (n native) header(single bool) (*gp.Header, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	token, err := gp.OpenToken(cache, "googleplay/token.json")
	if err != nil {
		return nil, err
	}
	device, err := gp.OpenDevice(cache, n.path)
	if err != nil {
		return nil, err
	}
	if single {
		return token.SingleAPK(device)
	}
	return token.Header(device)
}
