package repository

import (
	"database/sql"
	"net/mail"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mkideal/pkg/debug"
)

const (
	sqlCreateDatabase = "CREATE DATABASE IF NOT EXISTS `smtpd` ENGINE=InnoDB DEFAULT CHARSET=utf8"

	sqlUseDatabase = "USE smtpd"

	sqlCreateTableEmail = "CREATE TABLE IF NOT EXISTS email (" +
		"`id` INT NOT NULL AUTO_INCREMENT," +
		"`username` varchar(64) NOT NULL," +
		"`from` varchar(64) NOT NULL," +
		"`tos` text NOT NULL," +
		"`data` blob," +
		"PRIMARY KEY ( id )" +
		"FOREIGN KEY ( username ) REFERENCES mailbox ( username )" +
		")"

	sqlCreateTableMailbox = "CREATE TABLE IF NOT EXISTS mailbox(" +
		"`id` INT NOT NULL AUTO_INCREMENT," +
		"`username` varchar(64) NOT NULL," +
		"`address` varchar(64) NOT NULL," +
		"`create_date` varchar(32) NOT NULL," +
		"PRIMARY KEY ( id )" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8"

	sqlFindMailbox = `SELECT username,address FROM mailbox WHERE username=? OR address=?`

	sqlSaveEmail = "INSERT INTO email(`account`,`from`,`tos`,`data`) values(?,?,?,?)"
)

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
		sqlCreateDatabase,
		sqlUseDatabase,
		sqlCreateTableEmail,
		sqlCreateTableMailbox,
	); err != nil {
		return nil, err
	}
	return repo, nil
}

func (repo *MysqlRepository) FindMailbox(usernameOrAddress string) (*mail.Address, bool) {
	rows, err := repo.db.Query(sqlFindMailbox, usernameOrAddress, usernameOrAddress)
	if err != nil {
		debug.Debugf("Query %q error: %v", sqlFindMailbox, err)
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

func (repo *MysqlRepository) SaveEmail(addr *mail.Address, from, tos string, data []byte) error {
	if data == nil {
		data = []byte{}
	}

	repo.locker.Lock()
	defer repo.locker.Unlock()

	sqlStr := "insert into email(`account`,`from`,`tos`,`data`) values(?,?,?,?)"
	_, err := repo.db.Exec(sqlStr, addr.Name, from, tos, data)
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
