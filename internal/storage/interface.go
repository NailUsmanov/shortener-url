package storage

type Storage interface {
	Save(url string) (string, error)
	Get(key string) (string, error)
}
