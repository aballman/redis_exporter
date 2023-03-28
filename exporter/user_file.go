package exporter

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

// LoadUserFile reads the redis user file and returns the user map
func LoadUserFile(userFile string) (map[string]string, error) {
	res := make(map[string]string)

	log.Debugf("start load user file: %s", userFile)
	bytes, err := ioutil.ReadFile(userFile)
	if err != nil {
		log.Errorf("load user file failed: %s", err)
		return nil, err
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		log.Errorf("user file format error: %s", err)
		return nil, err
	}

	log.Errorf("Loaded %d entries from %s", len(res), userFile)
	for k := range res {
		log.Debugf("%s", k)
	}

	return res, nil
}

