package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

//Item how a neat thing is stored in memory
type Item struct {
	URL             string
	Description     string
	ShortLink       string //for links.ethohampton.com, just include the shortcut, should take precidence over url
	PostedOnTwitter bool
	AddDate         time.Time
}

//PublicItem What is sent to users via JSON api
type PublicItem struct {
	URL         string
	Description string
}

//ItemStorage the format how items are stored in the file
type ItemStorage struct {
	Items []Item
}

const masterJSONFile = "neatStuff.json"
const shortLinkBase = "https://links.ethohampton.com/"
const numToShowPublic = 5

var itemsToServe []PublicItem

func main() {

}

func serveJSON() {
	if len(itemsToServe) != numToShowPublic {
		its := loadItemsFromFile(masterJSONFile)
		itemsToServe = getLastNItemsAsPublic(its, numToShowPublic)
	}

	output, _ := json.Marshal(itemsToServe)
	fmt.Println(output)
}

func loadItemsFromFile(filename string) *ItemStorage {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	}

	var obj ItemStorage
	err = json.Unmarshal(data, &obj)
	if err != nil {
		fmt.Println(err)
	}

	return &obj
}

func storeItemsInFile(items *ItemStorage, filename string) {
	file, _ := json.Marshal(items)
	_ = ioutil.WriteFile(filename, file, 0644)
}

func getLastNItemsAsPublic(items *ItemStorage, n int) []PublicItem {
	length := len(items.Items)
	transform := items.Items[length-n-1:] // grab last n elements of items we are dealing with
	out := make([]PublicItem, n)

	//for each item, set the right URL (short or not) and description
	for i := 0; i < n; i++ {
		var newI PublicItem
		item := transform[i]
		if item.ShortLink != "" {
			newI.URL = shortLinkBase + item.ShortLink
		} else {
			newI.URL = item.URL
		}
		newI.Description = item.Description
		out[i] = newI
	}
	return out
}
