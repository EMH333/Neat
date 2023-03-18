package util

import (
	"encoding/json"
	"ethohampton.com/Neat/internal/types"
	"io/ioutil"
	"log"
)

func LoadItemsFromFile(filename string) *types.ContentStorage {
	//don't really care about errors in this method because golang will just return sane defaults if it can't parse or read
	data, _ := ioutil.ReadFile(filename)

	var obj types.ContentStorage
	err := json.Unmarshal(data, &obj)
	if err != nil {
		log.Fatal("Couldn't unmarshal JSON items")
		return nil
	}
	return &obj
}

func StoreItemsInFile(items *types.ContentStorage, filename string) {
	file, _ := json.Marshal(items)
	_ = ioutil.WriteFile(filename, file, 0644)
}
