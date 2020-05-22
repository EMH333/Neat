package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

const masterJSONFile = "./neatStuff.json"
const shortLinkBase = "https://links.ethohampton.com/"
const numToShowPublic = 5

const adminKeyPath = "./admin.key"

var adminKey string
var port = ":8080"

var itemsToServe []PublicItem

func main() {
	if len(os.Args) > 1 && isInteger(os.Args[1]) {
		port = ":" + os.Args[1]
	}
	log.Printf("Listening at http://localhost%s", port)

	adminKey = getAdminKey(adminKeyPath)

	http.HandleFunc("/json", serveJSON)
	http.HandleFunc("/add", serveAdd)
	http.HandleFunc("/", serveInfo)
	log.Fatal(http.ListenAndServe(port, nil))
}

func serveJSON(w http.ResponseWriter, r *http.Request) {
	if itemsToServe == nil || len(itemsToServe) != numToShowPublic {
		its := loadItemsFromFile(masterJSONFile)
		itemsToServe = getLastNItemsAsPublic(its, numToShowPublic)
	}

	output, _ := json.Marshal(itemsToServe)
	fmt.Fprint(w, string(output))
}

func serveAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "./static/add.html")
		return
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			//fmt.Printf("ParseForm() err: %v", err)
			w.WriteHeader(400)
			fmt.Fprint(w, "Some sort of form parsing error")
			return
		}

		description := r.FormValue("description")
		url := r.FormValue("url")

		if description == "" || url == "" {
			fmt.Fprint(w, "Please enter stuff in the form")
			return
		}

		//make sure there is permision to add something to the neat list
		key := r.FormValue("password")
		if key == "" || adminKey == "" || key != adminKey {
			fmt.Fprint(w, "Invalid password")
			return
		}

		//create new item, add to all items and delete currently cached items
		newItem := Item{Description: description, URL: url, AddDate: time.Now()}
		allItems := loadItemsFromFile(masterJSONFile)
		allItems.Items = append(allItems.Items, newItem)
		storeItemsInFile(allItems, masterJSONFile)
		itemsToServe = nil

		fmt.Fprint(w, "Success!")
	} else {
		fmt.Fprint(w, "Invalid Method")
	}
}

func serveInfo(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/info.html")
}

func loadItemsFromFile(filename string) *ItemStorage {
	//don't really care about errors in this method because golang will just return sane defaults if it can't parse or read
	data, _ := ioutil.ReadFile(filename)

	var obj ItemStorage
	json.Unmarshal(data, &obj)
	return &obj
}

func storeItemsInFile(items *ItemStorage, filename string) {
	file, _ := json.Marshal(items)
	_ = ioutil.WriteFile(filename, file, 0644)
}

func getLastNItemsAsPublic(items *ItemStorage, n int) []PublicItem {
	length := len(items.Items)
	if length < 0 {
		return make([]PublicItem, 0)
	}

	if length < n {
		n = length
	}

	transform := items.Items[length-n:] // grab last n elements of items we are dealing with
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

func getAdminKey(keyPath string) string {
	file, err := os.Open(keyPath)
	if err == nil {
		//log.Println(err)
		scanner := bufio.NewScanner(file)
		if scanner.Scan() {
			log.Println("Found admin key")
			return scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	defer file.Close()

	log.Fatal("Some error with the admin key, need key to add links")
	return "" //a empty string represents no admin account allowed
}

func isInteger(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
