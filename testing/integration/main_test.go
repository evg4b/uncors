package integration_test

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type TestCase struct {
	ExpectedPath string
	RequestPath  string
	ResponsePath string
}

type Connection struct {
	Request  *http.Request
	Response *http.Response
}

func ReadHTTPFromFile(r io.Reader) ([]Connection, error) {
	buf := bufio.NewReader(r)
	stream := make([]Connection, 0)

	for {
		req, err := http.ReadRequest(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return stream, err
		}

		resp, err := http.ReadResponse(buf, req)
		if err != nil {
			return stream, err
		}

		//save response body
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		resp.Body.Close()
		resp.Body = ioutil.NopCloser(b)

		stream = append(stream, Connection{Request: req, Response: resp})
	}
	return stream, nil

}

func TestDemo(t *testing.T) {
	files := make([]string, 0, 100)
	err := filepath.Walk("./tests/", func(path string, info os.FileInfo, err error) error {
		testutils.CheckNoError(t, err)

		if !info.IsDir() && (strings.HasSuffix(path, "_expected") || strings.HasSuffix(path, "_request") || strings.HasSuffix(path, "_response")) {
			files = append(files, path)
		}

		return nil
	})
	testutils.CheckNoError(t, err)

	print(files)

	demo := lo.GroupBy(files, func(item string) string {
		t1, _ := strings.CutSuffix(item, "_expected")
		t2, _ := strings.CutSuffix(t1, "_request")
		t3, _ := strings.CutSuffix(t2, "_response")

		return t3
	})

	for key, _ := range demo {
		t.Run(key, func(t *testing.T) {
			f, err := os.OpenFile(key+"_request", os.O_RDONLY, os.ModePerm)
			testutils.CheckNoError(t, err)
			defer f.Close()

			stream, err := ReadHTTPFromFile(f)
			if err != nil {
				log.Fatalln(err)
			}
			for _, c := range stream {
				b, err := httputil.DumpRequest(c.Request, true)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(b))
				b, err = httputil.DumpResponse(c.Response, true)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(string(b))
			}
		})
	}
	
	t.Run("demo", func(t *testing.T) {
		//f, err := os.Open("/Users/evg4b/Documents/uncors/demo.http")
		//if err != nil {
		//	panic(err)
		//}
		//
		//rrr := bufio.NewReader(f)
		//
		//r, err := http.ReadRequest(rrr)
		//if err != nil {
		//	panic(err)
		//}
		//
		//print(httputil.DumpRequest(r, true))
	})
}
