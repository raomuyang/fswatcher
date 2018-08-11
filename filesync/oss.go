package filesync

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"
	"strings"
)

type OSSFileSync struct {
	access Access
}

// 创建一个基于OSS文件同步工具
func NewOSSFileSync(access Access) OSSFileSync {
	ossfs := OSSFileSync{access: access}
	return ossfs
}

func (ossfs OSSFileSync) Put(localFile string, key string) (downloadURL string, err error) {
	log.Debugf("Put `%s` with key: `%s`", localFile, key)

	client, err := ossfs.initOSSClient()
	if err != nil {
		return
	}

	bucket, err := client.Bucket(ossfs.access.Bucket)
	if err != nil {
		return
	}

	err = bucket.PutObjectFromFile(key, localFile)
	if err != nil {
		return
	}

	if len(ossfs.access.Domain) > 0 {
		downloadURL = strings.Join([]string{ossfs.access.Domain, key}, "/")
	} else {
		domain := strings.Join([]string{bucket.BucketName, ossfs.access.Endpoint}, ".")
		downloadURL = strings.Join([]string{"https:/", domain, key}, "/")
	}
	return
}

func (ossfs OSSFileSync) Delete(key string) (err error) {
	log.Infof("try to remove remote object: %s", key)
	client, err := ossfs.initOSSClient()
	if err != nil {
		return
	}

	bucket, err := client.Bucket(ossfs.access.Bucket)
	if err != nil {
		return
	}

	bucket.DeleteObject(key)
	return
}

func (ossfs OSSFileSync) initOSSClient() (*oss.Client, error) {
	return oss.New(ossfs.access.Endpoint, ossfs.access.AccessKeyID, ossfs.access.AccessKeySecret)
}
