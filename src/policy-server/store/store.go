package store

import (
	"database/sql"
	"errors"
	"fmt"
	"policy-server/models"
)

const schema = `
CREATE TABLE IF NOT EXISTS groups (
	id SERIAL PRIMARY KEY,
	guid text
);

CREATE TABLE IF NOT EXISTS destinations (
	id SERIAL PRIMARY KEY,
	group_id int REFERENCES groups(id),
	port int,
	protocol text,
	UNIQUE (group_id, port, protocol)
);

CREATE TABLE IF NOT EXISTS policies (
	id SERIAL PRIMARY KEY,
	group_id int REFERENCES groups(id),
	destination_id int REFERENCES destinations(id),
	UNIQUE (group_id, destination_id)
);
`

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . Store
type Store interface {
	Create([]models.Policy) error
	All() ([]models.Policy, error)
}

//go:generate counterfeiter -o ../fakes/db.go --fake-name Db . db
type db interface {
	Begin() (*sql.Tx, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

//go:generate counterfeiter -o ../fakes/db.go --fake-name Db . db
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Commit() error
	Rollback() error
}

var RecordNotFoundError = errors.New("record not found")

type store struct {
	conn        db
	group       GroupCreator
	destination DestinationCreator
}

func NewGroup() (GroupCreator, error) {
	return &Group{}, nil
}

func NewDestination() (DestinationCreator, error) {
	return &Destination{}, nil
}

func New(dbConnectionPool db, g GroupCreator, d DestinationCreator) (Store, error) {
	err := setupTables(dbConnectionPool)
	if err != nil {
		return nil, fmt.Errorf("setting up tables: %s", err)
	}

	return &store{
		conn:        dbConnectionPool,
		group:       g,
		destination: d,
	}, nil
}

//go:generate counterfeiter -o ../fakes/group_creator.go --fake-name GroupCreator . GroupCreator
type GroupCreator interface {
	Create(Transaction, string) (int, error)
}

type Group struct {
	conn Transaction
}

//go:generate counterfeiter -o ../fakes/destination_creator.go --fake-name DestinationCreator . DestinationCreator
type DestinationCreator interface {
	Create(Transaction, int, int, string) (int, error)
}

type Destination struct {
	conn Transaction
}

func (g *Group) Create(trxn Transaction, guid string) (int, error) {
	_, err := trxn.Exec(
		`INSERT INTO groups (guid) SELECT $1
			WHERE
				NOT EXISTS (
					SELECT guid FROM groups WHERE guid = $1
				)
			`,
		guid,
	)
	if err != nil {
		return -1, err
	}

	var id int
	err = trxn.QueryRow(`SELECT id FROM groups WHERE guid = $1`, guid).Scan(&id)

	return id, err
}

func (d *Destination) Create(trxn Transaction, destination_group_id int, port int, protocol string) (int, error) {
	_, err := trxn.Exec(
		`INSERT INTO destinations (group_id, port, protocol)
				SELECT $1, $2, $3
				WHERE
					NOT EXISTS (
						SELECT *
						FROM destinations
						WHERE group_id = $1 AND port = $2 AND protocol = $3
					)`,
		destination_group_id,
		port,
		protocol,
	)
	if err != nil {
		return -1, err
	}

	var id int
	err = trxn.QueryRow(
		`SELECT id FROM destinations
				WHERE group_id = $1 AND port = $2 AND protocol = $3`,
		destination_group_id,
		port,
		protocol,
	).Scan(&id)

	return id, err
}

func (s *store) Create(policies []models.Policy) error {
	trxn, err := s.conn.Begin()
	if err != nil {
		panic(err)
	}

	for _, policy := range policies {
		source_group_id, err := s.group.Create(trxn, policy.Source.ID)
		if err != nil {
			txErr := trxn.Rollback()
			if txErr != nil {
				panic(txErr)
			}

			return fmt.Errorf("creating group: %s", err)
		}

		destination_group_id, err := s.group.Create(trxn, policy.Destination.ID)
		if err != nil {
			txErr := trxn.Rollback()
			if txErr != nil {
				panic(txErr)
			}

			return fmt.Errorf("creating group: %s", err)
		}

		destination_id, err := s.destination.Create(trxn, destination_group_id, policy.Destination.Port, policy.Destination.Protocol)
		if err != nil {
			txErr := trxn.Rollback()
			if txErr != nil {
				panic(txErr)
			}

			return fmt.Errorf("creating destination: %s", err)
		}

		_, err = trxn.Exec(
			`INSERT INTO policies (group_id, destination_id)
				SELECT $1, $2
				WHERE
					NOT EXISTS (
						SELECT *
						FROM policies
						WHERE group_id = $1 AND destination_id = $2
					)`,
			source_group_id,
			destination_id,
		)
		if err != nil {
			txErr := trxn.Rollback()
			if txErr != nil {
				panic(txErr)
			}

			panic(err)
		}
	}

	err = trxn.Commit()
	if err != nil {
		panic(err)
	}

	return nil
}

// func (s *store) Get() (models.Policy, error) {
// 	var container models.Container
// 	err := s.conn.Get(&container, "SELECT * FROM container WHERE id=$1", id)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return models.Container{}, RecordNotFoundError
// 		}
// 		return container, fmt.Errorf("getting record: %s", err)
// 	}

// return models.Policy{}, nil
// }

func (s *store) All() ([]models.Policy, error) {
	policies := []models.Policy{}

	rows, err := s.conn.Query(`
		select src_grp.guid,
				dst_grp.guid,
				destinations.port,
				destinations.protocol
			from policies
			left outer join groups as src_grp on (policies.group_id = src_grp.id)
			left outer join destinations on (destinations.id = policies.destination_id)
			left outer join groups as dst_grp on (destinations.group_id = dst_grp.id);`)
	if err != nil {
		return nil, fmt.Errorf("listing all: %s", err)
	}

	for rows.Next() {
		var source_id, destination_id, protocol string
		var port int
		err = rows.Scan(&source_id, &destination_id, &port, &protocol)
		if err != nil {
			return nil, fmt.Errorf("listing all: %s", err)
		}

		policies = append(policies, models.Policy{
			Source: models.Source{
				ID: source_id,
			},
			Destination: models.Destination{
				ID:       destination_id,
				Protocol: protocol,
				Port:     port,
			},
		})
	}

	return policies, nil
}

func setupTables(dbConnectionPool db) error {
	_, err := dbConnectionPool.Exec(schema)
	return err
}
