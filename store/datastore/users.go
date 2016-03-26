package datastore

import (
	"database/sql"

	"github.com/kusubooru/teian/teian"
)

func (db *datastore) GetUser(username string) (*teian.User, error) {
	var (
		u     teian.User
		pass  sql.NullString
		email sql.NullString
	)
	err := db.QueryRow(userGetQuery, username).Scan(
		&u.ID,
		&u.Name,
		&pass,
		&u.JoinDate,
		&u.Admin,
		&email,
	)
	if err != nil {
		return nil, err
	}
	if pass.Valid {
		u.Pass = pass.String
	}
	if email.Valid {
		u.Email = email.String
	}
	return &u, nil
}

const userGetQuery = `
SELECT * 
FROM users 
WHERE name = ?
`
