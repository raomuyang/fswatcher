package filesync

type Access struct {
	Endpoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Bucket          string `yaml:"bucket"`
	// 绑定的外链域名
	Domain string `yaml:"domain"`
}

type FileSync interface {
	Put(localFile string, key string) (downloadURL string, err error)
	Delete(key string) error
}
