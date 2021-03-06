package store

import (
	"fmt"
	"policy-server/db"
)

type DestinationMetadataTable struct{}

func (d *DestinationMetadataTable) Create(tx db.Transaction, terminalGUID, name, description string) (int64, error) {
	driver := tx.DriverName()
	if driver == "mysql" {
		result, err := tx.Exec(tx.Rebind(`
			INSERT INTO destination_metadatas (terminal_guid, name, description)
			VALUES (?, ?, ?)
		`),
			terminalGUID,
			name,
			description,
		)
		if err != nil {
			return -1, fmt.Errorf("failed to create destination metadata: %s", err)
		}
		return result.LastInsertId()
	} else if driver == "postgres" {
		var id int64

		err := tx.QueryRow(tx.Rebind(`
			INSERT INTO destination_metadatas (terminal_guid, name, description)
			VALUES (?,?,?)
			RETURNING id
		`),
			terminalGUID,
			name,
			description,
		).Scan(&id)

		if err != nil {
			return -1, fmt.Errorf("failed to create destination metadata: %s", err)
		}

		return id, nil

	}
	return -1, fmt.Errorf("unknown driver: %s", driver)
}
