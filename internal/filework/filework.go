package filework

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"main/internal/config"
	"net/http"
	"slices"
	"strings"
)

type Zip struct {
	Buf    bytes.Buffer
	Urls   map[string]string
	Env    config.Config
	Status *string
}

func PrepareZip(env config.Config) *Zip {
	status := fmt.Sprintf("Prepare zip, u can add another %d", env.MAX_FILES_IN_ZIP)
	return &Zip{Buf: bytes.Buffer{}, Urls: make(map[string]string), Env: env, Status: &status}
}

func (z *Zip) MakeZip() ([]byte, error) {
	zip := zip.NewWriter(&z.Buf)

	for name, url := range z.Urls {
		file, err := GetFileFromUrl(url, false)
		if err != nil {
			continue
		}
		err = addToZip(zip, name, file)
		if err != nil {
			return nil, fmt.Errorf("error: %v", err)
		}
	}
	err := zip.Close()
	if err != nil {
		return nil, fmt.Errorf("error close zip file: %v", err)
	}
	return z.Buf.Bytes(), nil
}

func (z *Zip) AppendUrl(url string) error {
	if len(z.Urls) >= z.Env.MAX_FILES_IN_ZIP {
		return fmt.Errorf("reached limit of files in zip archive")
	}
	split_url := strings.Split(url, "/")
	fileName := split_url[len(split_url)-1]
	extension := strings.Split(fileName, ".")
	if slices.Contains(z.Env.ALLOWED_EXTENSIONS, extension[1]) {
		_, ok := z.Urls[fileName]
		if ok {
			name := strings.ReplaceAll(fileName, "."+extension[1], fmt.Sprintf("-%d."+extension[1], len(z.Urls)+1))
			z.Urls[name] = url
			z.UpdateStatus()
			return nil
		}
		z.Urls[fileName] = url
		z.UpdateStatus()
		return nil
	}
	return fmt.Errorf("extension not allowed")
}

func (z *Zip) UpdateStatus() {
	*z.Status = fmt.Sprintf("Prepare zip, u can add another %d", z.Env.MAX_FILES_IN_ZIP-len(z.Urls))
}

func addToZip(zipWriter *zip.Writer, name string, file io.ReadCloser) error {
	defer file.Close()
	writer, err := zipWriter.Create(name)
	if err != nil {
		return fmt.Errorf("failed create file in zip: %v", err)
	}
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed write file in zip: %v", err)
	}
	return nil
}

func GetFileFromUrl(url string, check bool) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error get requests on url: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("bad ststus code: %d", resp.StatusCode)
	}
	if check {
		resp.Body.Close()
	}
	return resp.Body, nil
}
