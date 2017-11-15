// CopyRight Notice: All rights reserved
//
//

package dfc

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

var (
	httpClient *http.Client
)

const (
	MaxIdleConnections int = 20
	RequestTimeout     int = 5
)

const (
	IP   = "ip"
	PORT = "port"
	ID   = "id"
)

// createHTTPClient for connection re-use
func createHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

// Proxyhdlr function serves request coming to listening port of DFC's Proxy Client.
// It supports GET and POST method only and return 405 error non supported Methods.
func proxyhdlr(w http.ResponseWriter, r *http.Request) {

	glog.Infof("Proxy Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
	switch r.Method {
	case "GET":
		// Serve the resource.
		// TODO Give proper error if no server is registered and client is requesting data
		// or may be directly get from S3??
		if len(ctx.smap) < 1 {
			// No storage server is registered yet
			glog.Errorf("Storage Server count = %d  Proxy Request from %s: %s %q \n",
				len(ctx.smap), r.RemoteAddr, r.Method, r.URL)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)

		} else {

			sid := doHashfindServer(html.EscapeString(r.URL.Path))
			proxyclientRequest(sid, w, r)
		}

	case "POST":
		//Proxy server may will get POST for  storage server registration only
		r.ParseForm()
		glog.Infof("request content %s  \n", r.Form)
		var sinfo serverinfo
		// Parse POST values
		for str, val := range r.Form {
			if str == IP {
				//glog.Infof(" str is = %s val = %s \n", str, val)
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.ip = strings.Join(val, "")
			}
			if str == PORT {
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.port = strings.Join(val, "")
			}
			if str == ID {
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.id = strings.Join(val, "")
			}

		}

		// Insert into Map based on ID and fail if duplicates.
		// TODO Fail if there already client registered with same ID
		ctx.smap[sinfo.id] = sinfo
		glog.Infof(" IP = %s Port = %s  Id = %s Curlen of map = %d \n",
			sinfo.ip, sinfo.port, sinfo.id, len(ctx.smap))
		fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))

	case "PUT":
	case "DELETE":
	default:
		glog.Errorf("Invalid Proxy Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)

	}

}

// It registerss DFC's storage Server Instance with DFC's Proxy Client.
// A storage server uses ID, IP address and Port for registration with Proxy Client.
func registerwithproxy() error {
	httpClient = createHTTPClient()
	var err error
	// Proxy well known address
	proxyUrl := ctx.configparam.pcparam.pclienturl
	resource := "/"
	data := url.Values{}
	ipaddr := getipaddr()

	// Posting IP address, Port ID and ID as part of storage server registration.
	data.Set(IP, ipaddr)
	data.Add(PORT, string(ctx.configparam.lsparam.port))
	data.Add(ID, string(ctx.configparam.Id))

	u, _ := url.ParseRequestURI(string(proxyUrl))
	u.Path = resource
	urlStr := u.String()
	glog.Infof("Proxy URL : %s \n ", string(urlStr))

	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode()))
	if err != nil {
		glog.Errorf("Error Occured. %+v", err)
		return err
	}
	req.Header.Add("Authorization", "auth_token=\"XXXXXXX\"")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// Use httpClient to send request
	response, err := httpClient.Do(req)
	if err != nil && response == nil {
		glog.Errorf("Error sending request to Proxy server %+v \n", err)
		return err
	} else {
		// Close the connection to reuse it
		defer response.Body.Close()

		// Let's check if the work actually is done
		// Did we get 200 OK responsea?
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			glog.Errorf("Couldn't parse response body. %+v \n", err)
		}

		glog.Infof("Response Body: %s \n", string(body))
	}
	return err
}

// ProxyclientRequest submit a new http request to one of DFC storage server.
// Storage server ID is provided as one of argument to this call.
func proxyclientRequest(sid string, w http.ResponseWriter, r *http.Request) {
	glog.Infof(" Request path = %s Sid = %s Port = %s \n",
		html.EscapeString(r.URL.Path), sid, ctx.smap[sid].port)

	url := "http://" + ctx.smap[sid].ip + ":" + ctx.smap[sid].port + html.EscapeString(r.URL.Path)
	glog.Infof(" URL = %s \n", url)
	resp, err := http.Get(url)
	if err != nil {
		glog.Errorf("Failed to get url = %s err = %q", url, err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	glog.Infof(" URL = %s Response  = %s \n", url, body)
	fmt.Fprintf(w, "DFC-Daemon %q", html.EscapeString(r.URL.Path))
}
