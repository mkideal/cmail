package repository

import (
	"database/sql"
	"net/mail"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mkideal/pkg/debug"
)

const createTableEmail = "CREATE TABLE IF NOT EXISTS email (" +
	"`id` INT NOT NULL AUTO_INCREMENT," +
	"`from` varchar(64) NOT NULL," +
	"`tos` text NOT NULL," +
	"`data` blob," +
	"PRIMARY KEY ( id )" +
	")"

const createTableMailbox = "CREATE TABLE IF NOT EXISTS mailbox(" +
	"`id` INT NOT NULL AUTO_INCREMENT," +
	"`username` varchar(64) NOT NULL," +
	"`address` varchar(64) NOT NULL," +
	"`create_date` varchar(32) NOT NULL," +
	"PRIMARY KEY ( id )" +
	")"

type MysqlRepository struct {
	locker sync.Mutex
	db     *sql.DB
}

func Mysql(dbsource string) (*MysqlRepository, error) {
	db, err := sql.Open("mysql", dbsource)
	if err != nil {
		return nil, err
	}

	repo := new(MysqlRepository)
	repo.db = db
	if err := multiExec(db,
		"CREATE DATABASE IF NOT EXISTS `smtpd`",
		"USE smtpd",
		createTableEmail,
		createTableMailbox,
	); err != nil {
		return nil, err
	}
	return repo, nil
}

func (repo *MysqlRepository) FindMailbox(usernameOrAddress string) (*mail.Address, bool) {
	sqlStr := `select username,address from mailbox where username="` +
		usernameOrAddress + `" or address="` + usernameOrAddress + `"`
	rows, err := repo.db.Query(sqlStr)
	if err != nil {
		debug.Debugf("Query %q error: %v", sqlStr, err)
		return nil, false
	}
	addr := &mail.Address{}
	if rows.Next() {
		if err := rows.Scan(&addr.Name, &addr.Address); err != nil {
			debug.Debugf("Scan result error: %v", err)
			return nil, false
		}
		return addr, true
	}
	return nil, false
}

func (repo *MysqlRepository) SaveEmail(from, tos string, data []byte) error {
	if data == nil {
		data = []byte{}
	}

	repo.locker.Lock()
	defer repo.locker.Unlock()

	sqlStr := "insert into email(`from`,`tos`,`data`) values(?,?,?)"
	_, err := repo.db.Exec(sqlStr, from, tos, data)
	if err != nil {
		debug.Debugf("SaveEmail error: %v", err)
	}
	return err
}

func multiExec(db *sql.DB, sqls ...string) error {
	for _, sql := range sqls {
		if _, err := db.Exec(sql); err != nil {
			return err
		}
	}
	return nil
}
