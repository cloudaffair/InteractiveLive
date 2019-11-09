package common

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"
	"fmt"
)

var ctxHttpClient        = "httpClient"
var g_client *http.Client
var g_client_long *http.Client

func init() {
	tr := &http.Transport{DisableKeepAlives: true}
	g_client = &http.Client{Transport: tr}
	g_client.Timeout = 300 * time.Second

	trl := &http.Transport{DisableKeepAlives: true}
	g_client_long = &http.Client{Transport: trl}
	g_client_long.Timeout = 900 * time.Second
}


func HttpSubmitData(ctx context.Context, mode string, url string, headers *http.Header, data []byte) ([]byte, *http.Response, error) {
	CheckContext(ctx)

	ioReader := bytes.NewReader(data)
	url = AddHttpSchemeIfMissing(url)
	req, err := http.NewRequest(mode, url, ioReader)
	if err != nil {
		httpErr := HttpErrorf(http.StatusInternalServerError, "Submit method=%s url=%s Http.NewRequest failed: %s", mode, url, err.Error())
		httpErr.SetCallerDepth(ErrorCallerDepth + 2)
		return nil, nil, httpErr
	}
	req.ContentLength = int64(len(data))

	if headers != nil {
		for key, value := range *headers {
			var multiVal string
			for _, part := range value {
				multiVal = multiVal + part + ","
			}
			formattedValue := multiVal[:len(multiVal)-1] // strip off last comma
			req.Header.Set(key, formattedValue)
		}
	}

	resp, err := doTimedRequest(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		httpErr := HttpErrorf(resp.StatusCode, "Submit method=%s url=%s ioUtil.ReadAll failed: %s", mode, url, err.Error())
		httpErr.SetCallerDepth(ErrorCallerDepth + 2)
		return body, resp, httpErr
	}

	if IsStatusCodeBad(resp.StatusCode) {
		httpErr := HttpErrorf(resp.StatusCode, "Submit method=%s url=%s failed with rc=%d, req=%s, resp=%s", mode, url, resp.StatusCode, string(data), string(body))
		httpErr.SetCallerDepth(ErrorCallerDepth + 2)
		return body, resp, httpErr
	}
	return body, resp, nil
}

func HttpClient(ctx context.Context) (*http.Client, error) {
	if ctx != nil {
		if httpClient, ok := ctx.Value(ctxHttpClient).(*http.Client); ok {
			return httpClient, nil
		}
	}
	return nil, fmt.Errorf("no httpClient in context")
}

func doTimedRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	CheckContext(ctx)
	var statusCode int = http.StatusGatewayTimeout

	client, err := HttpClient(ctx)

	if err != nil {
		if req.Method == "POST" {
			client = g_client_long
		} else {
			client = g_client
		}
	}

	req.Close = true

	if ctx != nil {
		req = req.WithContext(ctx)
	}
	fmt.Println("Request fired as: ", req)
	resp, httpErr := client.Do(req)

	if httpErr == nil {
		statusCode = resp.StatusCode
	}

	if httpErr != nil {
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
			_, _ = io.Copy(ioutil.Discard, resp.Body)
		}

		httpErr := HttpErrorf(statusCode, "Submit method=%s url=%s failed: %s", req.Method, req.URL.String(), httpErr.Error())
		httpErr.SetCallerDepth(ErrorCallerDepth + 3)

		return nil, httpErr
	}
	return resp, nil
}

func IsStatusCodeBad(code int) bool {
	return code >= http.StatusBadRequest || code < http.StatusContinue
}

func AddHttpSchemeIfMissing(url string) (httpUrl string) {
	if !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		httpUrl = "http://" + url
	} else {
		httpUrl = url
	}
	return httpUrl
}

