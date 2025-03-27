package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "embed"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/example/helpers"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

//go:embed web/index.html
var page []byte

//go:embed wasm/zig-out/bin/pastesec.wasm
var wasm_bin []byte

const max_paste_size = 16 * 1024

func UploadPaste(ctx context.Context, client objectstorage.ObjectStorageClient, data []byte, namespaceName, bucketName, objectName string) error {
	req := objectstorage.PutObjectRequest{
		NamespaceName: common.String(namespaceName),
		ObjectName:    common.String(objectName),
		PutObjectBody: io.NopCloser(bytes.NewReader(data)),
		BucketName:    common.String(bucketName)}

	_, err := client.PutObject(ctx, req)
	return err
}

func DownloadPaste(ctx context.Context, client objectstorage.ObjectStorageClient, namespaceName, bucketName, objectName string) ([]byte, error) {
	req := objectstorage.GetObjectRequest{
		NamespaceName: common.String(namespaceName),
		ObjectName:    common.String(objectName),
		BucketName:    common.String(bucketName)}

	resp, err := client.GetObject(ctx, req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Content)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func main() {
	ctx := context.Background()

	namespaceName, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		log.Fatalln("NAMESPACE environment variable is not set")
	}

	bucketName, ok := os.LookupEnv("BUCKET")
	if !ok {
		log.Fatalln("BUCKET environment variable is not set")
	}

	client, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(common.DefaultConfigProvider())
	helpers.FatalIfError(err)

	http.HandleFunc("POST /upload", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(io.LimitReader(r.Body, max_paste_size))
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("400 Bad Request"))
			return
		}

		md5sum := md5.Sum(body)
		objectKey := hex.EncodeToString(md5sum[:])
		err = UploadPaste(ctx, client, body, namespaceName, bucketName, objectKey)
		if err != nil {
			log.Println(err)

			w.WriteHeader(500)
			w.Write([]byte("500 Internal Server Error"))
			return
		}

		w.Write([]byte(objectKey))
	})

	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			w.WriteHeader(404)
			w.Write([]byte("404 Page Not Found"))
			return
		}

		paste, err := DownloadPaste(ctx, client, namespaceName, bucketName, id)
		if err != nil {
			w.WriteHeader(404)
			w.Write([]byte("404 Page Not Found"))
			return
		}

		w.Write(paste)
	})

	http.HandleFunc("/wasm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")
		w.Write(wasm_bin)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(page)
	})

	fmt.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
