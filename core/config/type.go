package config

type ApiUser struct {
	ApiSign string `json:"api_sign,omitempty" structs:"api_sign,omitempty"`
	ExName  string `json:"exchange,omitempty" structs:"exchange,omitempty"`
	ExType  string `json:"ex_type,omitempty" structs:"exchange,omitempty"`
	Uid     string `json:"uid,omitempty" structs:"name,omitempty"`
	Key     string `json:"key,omitempty" structs:"key,omitempty"`
	Secret  string `json:"secret,omitempty" structs:"secret,omitempty"`
}

type MysqlUser struct {
	Host     string `json:"host,omitempty" structs:"host,omitempty"`
	User     string `json:"user,omitempty" structs:"user,omitempty"`
	Password string `json:"password,omitempty" structs:"password,omitempty"`
	Port     string `json:"port,omitempty" structs:"port,omitempty"`
	Database string `json:"database,omitempty" structs:"database,omitempty"`
	Charset  string `json:"charset,omitempty" structs:"charset,omitempty"`
}

type RedisUser struct {
	Host     string `json:"host,omitempty" structs:"host,omitempty"`
	Port     string `json:"port,omitempty" structs:"port,omitempty"`
	User     string `json:"user,omitempty" structs:"user,omitempty"`
	Password string `json:"password,omitempty" structs:"password,omitempty"`
	Db       int    `json:"database,omitempty,string" structs:"database,omitempty,string"`
	Tls      bool   `json:"tls,omitempty,bool" structs:"tls,omitempty,bool"`
}
