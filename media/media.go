package media

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"

	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
)

type Store struct {
	repo database.Repository[models.Blob]
}

func NewStore(repo database.Repository[models.Blob]) *Store {
	return &Store{repo: repo}
}

func (s *Store) Save(mediaType string, data []byte) (*models.Blob, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	blob := &models.Blob{ID: id, MediaType: mediaType, Data: base64.StdEncoding.EncodeToString(data)}
	if err := s.repo.Create(blob); err != nil {
		return nil, err
	}
	return blob, nil
}

func (s *Store) Decode(blob *models.Blob) ([]byte, error) {
	return base64.StdEncoding.DecodeString(blob.Data)
}

func (s *Store) Get(id string) (*models.Blob, error) {
	blobs, err := s.repo.Where("id = ?", id)
	if err != nil || len(blobs) == 0 {
		return nil, err
	}
	return &blobs[0], nil
}

func PublicURL(baseURL, id string) string {
	return strings.TrimRight(baseURL, "/") + "/media/" + id
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
