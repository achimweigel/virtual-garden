package virtualgarden

import (
	"fmt"
	"os"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func TestAliCloudBackup(t *testing.T) {
	// init
	// set the id and key to activate the test
	accessKeyId := ""
	accessSecretKey := ""

	if accessKeyId == "" || accessSecretKey == "" {
		return
	}

	endpoint := "oss-eu-central-1.aliyuncs.com"

	// Create an OSSClient instance.
	client, err := oss.New(endpoint, accessKeyId, accessSecretKey)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	listAliCloudBuckets(client)

	// create bucket
	bucketName := "dfkjhdfskhdfsdhfskduhdfkh"
	// Create a bucket (the default storage class is Standard) and set the ACL of the bucket to public read (the default ACL is private).
	err = client.CreateBucket(bucketName, oss.ACL(oss.ACLPrivate))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	listAliCloudBuckets(client)

}

func listAliCloudBuckets(client *oss.Client) {
	marker := ""
	for {
		lsRes, err := client.ListBuckets(oss.Marker(marker))
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(-1)
		}

		// By default, 100 buckets are listed each time.
		for _, bucket := range lsRes.Buckets {
			fmt.Println("Bucket: ", bucket.Name)
		}

		if lsRes.IsTruncated {
			marker = lsRes.NextMarker
		} else {
			break
		}
	}
}