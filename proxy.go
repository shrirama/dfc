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

func proxyhdlr(w http.ResponseWriter, r *http.Request) {

	glog.Infof("Proxy Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
	switch r.Method {
	case "GET":
		// Serve the resource.
		// TODO Give proper error if no server is registered and client is requesting data
		// or may be directly get from S3??

		sid := doHashfindServer(html.EscapeString(r.URL.Path))
		proxyclientRequest(sid, w, r)

	case "POST":
		//Proxy server may will get POST for  registration only
		// Need ParseForm method, to get data from form
		r.ParseForm()
		// attention: If you do not call ParseForm method, the following data can not be obtained form
		glog.Infof("request content %s  \n", r.Form) // print information on server side.
		var sinfo serverinfo
		for str, val := range r.Form {
			if str == "ip" {
				//glog.Infof(" str is = %s val = %s \n", str, val)
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.ip = strings.Join(val, "")
			}
			if str == "port" {
				glog.Infof("val : %s \n", strings.Join(val, ""))
				sinfo.port = strings.Join(val, "")
			}
			if str == "id" {
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
		// Update an existing record.
	case "DELETE":
		// Remove the record.
	default:
		// Give an error message.
	}

	glog.Infof("Proxy Request from %s: %s %q \n", r.RemoteAddr, r.Method, r.URL)
}

func registerwithproxy() error {
	httpClient = createHTTPClient()
	var err error
	// Proxy well known address
	proxyUrl := "http://localhost:8080"
	resource := "/"
	data := url.Values{}
	ipaddr := getipaddr()
	data.Set("ip", ipaddr)
	data.Add("port", string(ctx.configparam.lsparam.port))
	data.Add("id", string(ctx.configparam.Id))

	u, _ := url.ParseRequestURI(proxyUrl)
	u.Path = resource
	urlStr := u.String() // "http://api.com/user/"
	glog.Infof("Proxy URL : %s \n ", string(urlStr))

	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(data.Encode())) // <-- URL-encoded payload
	if err != nil {
		glog.Errorf("Error Occured. %+v", err)
		return err
	}
	req.Header.Add("Authorization", "auth_token=\"XXXXXXX\"")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	// use httpClient to send request
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
