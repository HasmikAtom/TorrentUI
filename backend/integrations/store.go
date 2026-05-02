package integrations

import (
	"database/sql"

	"github.com/hasmikatom/torrent/db"
)

type UserIntegrations struct {
	UserID      string `json:"userId" db:"user_id"`
	PlexEnabled bool   `json:"plexEnabled" db:"plex_enabled"`
	PlexToken   string `json:"plexToken,omitempty" db:"plex_token"`
	CreatedAt   int64  `json:"createdAt" db:"created_at"`
	UpdatedAt   int64  `json:"updatedAt" db:"updated_at"`
}

type Store struct {
	db *db.Database
}

func NewStore(d *db.Database) *Store {
	return &Store{db: d}
}

func (s *Store) GetIntegrations(userID string) (UserIntegrations, error) {
	var row UserIntegrations
	err := s.db.DB.Get(&row, "SELECT * FROM user_integrations WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		return UserIntegrations{UserID: userID}, nil
	}
	return row, err
}

func (s *Store) UpsertPlex(userID, token string, enabled bool) error {
	_, err := s.db.DB.Exec(`
		INSERT INTO user_integrations (user_id, plex_token, plex_enabled, updated_at)
		VALUES (?, ?, ?, unixepoch())
		ON CONFLICT(user_id) DO UPDATE SET
			plex_token   = excluded.plex_token,
			plex_enabled = excluded.plex_enabled,
			updated_at   = unixepoch()
	`, userID, token, enabled)
	return err
}

func (s *Store) SetPlexEnabled(userID string, enabled bool) error {
	_, err := s.db.DB.Exec(`
		UPDATE user_integrations
		SET plex_enabled = ?, updated_at = unixepoch()
		WHERE user_id = ?
	`, enabled, userID)
	return err
}

func (s *Store) DeletePlex(userID string) error {
	_, err := s.db.DB.Exec(`
		UPDATE user_integrations
		SET plex_token = '', plex_enabled = 0, updated_at = unixepoch()
		WHERE user_id = ?
	`, userID)
	return err
}
