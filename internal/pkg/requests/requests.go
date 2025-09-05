package requests

import (
	"bytes"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"
)

func SimpleGetCli(cli *fasthttp.Client, url string) (int, []byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	req.SetTimeout(1 * time.Minute)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := cli.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s url: %s\n", err, url)
		return -1, nil, err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, resp.StatusCode())
		return resp.StatusCode(), nil, nil
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		fmt.Println("Unzipping...")
		body, _ = resp.BodyGunzip()
	} else {
		body = resp.Body()
	}
	return resp.StatusCode(), body, nil
}

func SimpleGet(url string) (int, []byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.SetUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")
	req.SetTimeout(1 * time.Minute)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s url: %s\n", err, url)
		return -1, nil, err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		fmt.Printf("Expected status code %d but got %d\n", fasthttp.StatusOK, resp.StatusCode())
		return resp.StatusCode(), nil, nil
	}

	contentEncoding := resp.Header.Peek("Content-Encoding")
	var body []byte
	if bytes.EqualFold(contentEncoding, []byte("gzip")) {
		fmt.Println("Unzipping...")
		body, _ = resp.BodyGunzip()
	} else {
		body = resp.Body()
	}
	return resp.StatusCode(), body, nil
}
