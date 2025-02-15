package compression

import (
	"bytes"
	"encoding/base64"
	"github.com/andybalholm/brotli"
	"io/ioutil"
	"net/url"
	"strings"
)

// DecodeBase64 decodes a base64 string without compression
func DecodeBase64(base64String string) (sourceJson string, ok bool) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// DecompressBrotliDecode decompresses a brotli encoded string a that is also base64 encoded
func DecompressBrotliDecode(base64String string) (sourceJson string, ok bool) {
	data, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return "", false
	}
	reader := bytes.NewReader(data)
	br := brotli.NewReader(reader)
	decompressedData, err := ioutil.ReadAll(br)
	if err != nil {
		return "", false
	}
	return string(decompressedData), true
}

func JsonFromEncodedUrl(urlString string) (sourceJson string, ok bool) {
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return "", false
	}
	zdata := parsedUrl.Query().Get("z") // look for zipped data first
	if zdata == "" {
		bdata := parsedUrl.Query().Get("b") // check for base64 encoded data
		if bdata == "" {
			return "", false
		}
		// REVIEW: do we really need to replace all spaces with +?
		return DecodeBase64(bdata) // strings.ReplaceAll(bdata, " ", "+"))
	} else {
		return DecompressBrotliDecode(strings.ReplaceAll(zdata, " ", "+"))
	}
}

func CompressBrotliEncode(fileData []byte) (base64String string, ok bool) {
	var buffer bytes.Buffer
	bw := brotli.NewWriter(&buffer)
	_, err := bw.Write(fileData)
	if err != nil {
		return "", false
	}
	err = bw.Close()
	if err != nil {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), true
}
