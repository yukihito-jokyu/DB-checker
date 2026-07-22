package domain

import (
	"errors"
	"strings"
)

var (
	ErrInvalidProfile       = errors.New("invalid profile")
	ErrInvalidActiveProfile = errors.New("invalid active profile")
)

type DBType string

const (
	DBTypeMySQL    DBType = "mysql"
	DBTypePostgres DBType = "postgres"
)

type Profile struct {
	ID       string
	Name     string
	DBType   DBType
	Host     string
	Port     int
	Database string
	Schema   string
	User     string
}

// プロファイル生成
func NewProfile(id, name string, dbType DBType, host string, port int, database, schema, user string) (Profile, error) {
	if id == "" || !validProfileName(name) || !validDBType(dbType) {
		return Profile{}, ErrInvalidProfile
	}

	if !validConnectionValue(host) || !validPort(port) || !validConnectionValue(database) || !validConnectionValue(user) || !validSchema(dbType, schema) {
		return Profile{}, ErrInvalidProfile
	}

	return Profile{ID: id, Name: name, DBType: dbType, Host: host, Port: port, Database: database, Schema: schema, User: user}, nil
}

// アクティブプロファイル検証
func ValidateActiveProfile(profiles []Profile, activeID *string) error {
	if activeID == nil {
		return nil
	}
	for _, profile := range profiles {
		if profile.ID == *activeID {
			return nil
		}
	}

	return ErrInvalidActiveProfile
}

// プロファイル名検証
func validProfileName(name string) bool {
	return strings.TrimSpace(name) != "" && len([]rune(name)) <= 100
}

// データベース種別検証
func validDBType(dbType DBType) bool {
	return dbType == DBTypeMySQL || dbType == DBTypePostgres
}

// 接続値検証
func validConnectionValue(value string) bool {
	if value == "" || len(value) > 100 {
		return false
	}
	for _, character := range value {
		if character < '!' || character > '~' {
			return false
		}
	}

	return true
}

// ポート番号検証
func validPort(port int) bool {
	return port >= 1 && port <= 65535
}

// スキーマ検証
func validSchema(dbType DBType, schema string) bool {
	if dbType == DBTypeMySQL {
		return schema == ""
	}

	return validConnectionValue(schema)
}
