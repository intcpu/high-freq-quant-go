package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func GetServerHost() string {
	host, err := os.Hostname()
	if err != nil {
		fmt.Errorf("can't getLocalNode %v", err)
		return ""
	}
	return host
}

func GetKeysConfig() map[string]ApiUser {
	var err error
	apiKeys := map[string]ApiUser{}
	if err = LoadJSON("../configs/api_keys.json", &apiKeys); err != nil {
		fmt.Errorf("can't load api_keys file %v", err)
		panic("can't load api_keys file")
	}
	return apiKeys
}

func GetMysqlConfig(database string) *MysqlUser {
	var err error
	keys := map[string]MysqlUser{}
	if err = LoadJSON("../configs/mysql_config.json", &keys); err != nil {
		fmt.Errorf("can't load mysql_config file %v", err)
		panic("can't load mysql_config file")
	}
	if _, ok := keys[database]; !ok {
		fmt.Errorf("can't select mysql_config database" + database)
		panic("can't select mysql_config database" + database)
	}
	dataKey := keys[database]
	return &dataKey
}

func GetRedisConfig(database string) *RedisUser {
	var err error
	keys := map[string]RedisUser{}
	if err = LoadJSON("../configs/redis_config.json", &keys); err != nil {
		fmt.Errorf("can't load redis_config file %v", err)
		panic("can't load redis_config file")
	}
	if _, ok := keys[database]; !ok {
		fmt.Errorf("can't select redis_config database" + database)
		panic("can't select redis_config database" + database)
	}
	dataKey := keys[database]
	return &dataKey
}

// LoadJSON reads the given file and unmarshals its content.
func LoadJSON(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}
