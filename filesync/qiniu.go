package filesync

import (
	"context"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	log "github.com/sirupsen/logrus"
	"strings"
)

type QiniuFileSync struct {
	access        Access
	mac           *qbox.Mac
	cfg           storage.Config
	bucketManager *storage.BucketManager
}

// 创建一个基于七牛云的文件存储工具
func NewQiniuFileSync(access Access) (qiniu QiniuFileSync) {

	qiniu.access = access

	bucket := access.Bucket
	endpoint := access.Endpoint
	accessKey := access.AccessKeyID
	secretKey := access.AccessKeySecret

	qiniu.mac = qbox.NewMac(accessKey, secretKey)

	qiniu.cfg = storage.Config{}
	if endpoint == "huadong" {
		qiniu.cfg.Zone = &storage.ZoneHuadong
	} else if endpoint == "huanan" {
		qiniu.cfg.Zone = &storage.ZoneHuanan
	} else if endpoint == "huabei" {
		qiniu.cfg.Zone = &storage.ZoneHuabei
	} else if endpoint == "beimei" {
		qiniu.cfg.Zone = &storage.ZoneBeimei
	} else {
		zone, err := storage.GetZone(accessKey, bucket)
		if err == nil {
			qiniu.cfg.Zone = zone
		}
	}

	qiniu.bucketManager = storage.NewBucketManager(qiniu.mac, &qiniu.cfg)
	return
}

func (qiniu QiniuFileSync) Put(localFile string, key string) (downloadURL string, err error) {

	log.Debugf("Put `%s` with key: `%s`", localFile, key)

	token, err := qiniu.initToken()

	formUploader := storage.NewFormUploader(&qiniu.cfg)
	ret := storage.PutRet{}

	putExtra := storage.PutExtra{}

	err = formUploader.PutFile(context.Background(), &ret, token, key, localFile, &putExtra)

	return strings.Join([]string{qiniu.access.Domain, key}, "/"), err
}

func (qiniu QiniuFileSync) Delete(key string) error {
	log.Infof("try to remove remote object: %s", key)
	return qiniu.bucketManager.Delete(qiniu.access.Bucket, key)
}

func (qiniu QiniuFileSync) initToken() (token string, err error) {
	putPolicy := storage.PutPolicy{Scope: qiniu.access.Bucket}
	putPolicy.Expires = 3600
	token = putPolicy.UploadToken(qiniu.mac)
	return
}
