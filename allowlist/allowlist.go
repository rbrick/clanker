package allowlist

import (
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
)

type Allowlist struct {
	repo database.Repository[models.AllowlistEntry]
}

func (a *Allowlist) AddEntry(entry *models.AllowlistEntry) error {
	return a.repo.Create(entry)
}

func (a *Allowlist) RemoveEntry(platform, userID string) error {
	entries, err := a.repo.Where("platform = ? AND user_id = ?", platform, userID)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil // No entry to delete
	}
	return a.repo.Delete(entries[0].ID)
}

func (a *Allowlist) IsAllowed(platform, userID string) (bool, error) {
	entries, err := a.repo.Where("platform = ? AND user_id = ?", platform, userID)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func NewAllowlist(repo database.Repository[models.AllowlistEntry]) *Allowlist {
	return &Allowlist{repo: repo}
}
