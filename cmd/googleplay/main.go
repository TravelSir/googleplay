package main

import (
	"flag"
	"fmt"
	gp "github.com/TravelSir/googleplay"
	"strings"
)

func main() {
	// a
	var app string
	flag.StringVar(&app, "a", "", "app")
	// arm64
	var arm64 bool
	flag.BoolVar(&arm64, "arm64", false, "arm64-v8a")
	// armeabi
	var armeabi bool
	flag.BoolVar(&armeabi, "armeabi", false, "armeabi-v7a")
	// d
	var device bool
	flag.BoolVar(&device, "d", false, "create device")
	// e
	var email string
	flag.StringVar(&email, "e", "", "email")
	// p
	var password string
	flag.StringVar(&password, "p", "", "password")
	// 代理
	var s5addr string
	flag.StringVar(&proxy, "s5addr", "", "s5addr")
	var s5user string
	flag.StringVar(&proxy, "s5addr", "", "s5user")
	var s5pwd string
	flag.StringVar(&proxy, "s5addr", "", "s5pwd")
	// purchase
	var (
		buf      strings.Builder
		purchase bool
	)
	buf.WriteString("Purchase app. ")
	buf.WriteString("Only needs to be done once per Google account.")
	flag.BoolVar(&purchase, "purchase", false, buf.String())
	// s
	var single bool
	flag.BoolVar(&single, "s", false, "single APK")
	// v
	var version uint64
	flag.Uint64Var(&version, "v", 0, "version")
	// verbose
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "dump requests")
	flag.Parse()
	if verbose {
		gp.LogLevel = 1
	}
	if email != "" {
		err := doToken(email, password, s5addr, s5user, s5pwd)
		if err != nil {
			panic(err)
		}
	} else {
		nat := newNative(armeabi, arm64)
		if device {
			err := nat.device()
			if err != nil {
				panic(err)
			}
		} else if app != "" {
			head, err := nat.header(single)
			if err != nil {
				panic(err)
			}
			if purchase {
				err := head.Purchase(app)
				if err != nil {
					panic(err)
				}
			} else if version >= 1 {
				err := doDelivery(head, app, version)
				if err != nil {
					panic(err)
				}
			} else {
				det, err := head.Details(app)
				if err != nil {
					panic(err)
				}
				fmt.Println(det)
			}
		} else {
			flag.Usage()
		}
	}
}
