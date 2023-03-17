package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

//Link how a neat thing is stored in memory
type Link struct {
	URL             string
	Description     string
	ShortLink       string //for links.ethohampton.com, just include the shortcut, should take precedence over url
	PostedOnTwitter bool
	AddDate         time.Time
}

//Short A piece of short content that can (eventually, once implemented) disappear
//Note: this contains some non-public info, but since this is displayed via template, that is okay
type Short struct {
	Title       string
	Content     string
	ID          string    // system generated, for admin use (but displayed on page)
	ReleaseDate time.Time // allow for delayed releasing, if before AddDate, then immediate release
	Pinned      bool      //TODO allow pinning shorts so they dont disappear
	Kept        uint64    //TODO allow anonymous users to "keep" a short so the time they are visible is extended
	AddDate     time.Time
}

//PublicLink What is sent to users via JSON api
type PublicLink struct {
	URL         string
	Description string
}

//ContentStorage the format how items are stored in the file
type ContentStorage struct {
	Links                   []Link  `json:"links"`
	Shorts                  []Short `json:"shorts"`
	ShortVisibilityDuration uint64  `json:"shortVisibilityDuration"`
}

const masterJSONFile = "./neatStuff.json"
const shortLinkBase = "https://links.ethohampton.com/"
const numToShowPublic = 5

const adminKeyPath = "./admin.key"

var adminKey string
var port = ":8080"

//These are all "caches" in that they can handle being empty
var itemsToServe []PublicLink

func main() {
	if len(os.Args) > 1 && isInteger(os.Args[1]) {
		port = ":" + os.Args[1]
	}
	log.Printf("Listening at http://localhost%s", port)

	adminKey = getAdminKey(adminKeyPath)

	http.HandleFunc("/json", serveJSON)
	http.HandleFunc("/add", serveAdd)
	http.HandleFunc("/all", serveAll)
	http.HandleFunc("/", serveInfo)
	log.Fatal(http.ListenAndServe(port, nil))
}

func serveJSON(w http.ResponseWriter, _ *http.Request) {
	if itemsToServe == nil || len(itemsToServe) != numToShowPublic {
		its := loadItemsFromFile(masterJSONFile)
		itemsToServe = getLastNItemsAsPublic(its, numToShowPublic)
	}

	output, _ := json.Marshal(itemsToServe)
	returnString(w, string(output))
	w.Header().Set("Content-Type", " application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "./static/add.html")
		return
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			//fmt.Printf("ParseForm() err: %v", err)
			w.WriteHeader(400)
			returnString(w, "Some sort of form parsing error")
			return
		}

		//make sure there is permission to add something to the neat list
		key := r.FormValue("password")
		if !correctKey(key) {
			returnString(w, "Invalid password")
			return
		}

		//for links
		{
			description := r.FormValue("description")
			url := r.FormValue("url")

			if description == "" || url == "" {
				returnString(w, "Please enter stuff in the form")
				return
			}

			//create new item, add to all items and delete currently cached items
			newItem := Link{Description: description, URL: url, AddDate: time.Now()}
			allItems := loadItemsFromFile(masterJSONFile)
			allItems.Links = append(allItems.Links, newItem)
			storeItemsInFile(allItems, masterJSONFile)
			itemsToServe = nil
		}

		returnString(w, "Success!")
		log.Println("User added a link")
	} else {
		w.WriteHeader(400)
		returnString(w, "Invalid Method")
	}
}

func serveAll(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			//fmt.Printf("ParseForm() err: %v", err)
			w.WriteHeader(400)
			returnString(w, "Some sort of form parsing error")
			return
		}

		//make sure there is permission to get all items
		key := r.FormValue("password")
		if !correctKey(key) {
			returnString(w, "Invalid password")
			return
		}

		allItems := loadItemsFromFile(masterJSONFile)
		output, _ := json.Marshal(allItems)
		returnString(w, string(output))
		log.Println("User accessed all links")
	} else {
		w.WriteHeader(400)
		returnString(w, "Invalid Method")
	}
}

func serveInfo(w http.ResponseWriter, r *http.Request) {
	//Send 404's for all pages that aren't the home page
	if r.URL.Path != "/" {
		w.WriteHeader(404)
		returnString(w, "404 Page Not Found")
		return
	}
	http.ServeFile(w, r, "./static/info.html")
}

func loadItemsFromFile(filename string) *ContentStorage {
	//don't really care about errors in this method because golang will just return sane defaults if it can't parse or read
	data, _ := ioutil.ReadFile(filename)

	var obj ContentStorage
	err := json.Unmarshal(data, &obj)
	if err != nil {
		log.Fatal("Couldn't unmarshal JSON items")
		return nil
	}
	return &obj
}

func storeItemsInFile(items *ContentStorage, filename string) {
	file, _ := json.Marshal(items)
	_ = ioutil.WriteFile(filename, file, 0644)
}

func getLastNItemsAsPublic(items *ContentStorage, n int) []PublicLink {
	length := len(items.Links)
	if length < 0 {
		return make([]PublicLink, 0)
	}

	if length < n {
		n = length
	}

	transform := items.Links[length-n:] // grab last n elements of items we are dealing with
	out := make([]PublicLink, n)

	//for each item, set the right URL (short or not) and description
	for i := 0; i < n; i++ {
		var newI PublicLink
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Fatal("Couldn't close file")
		}
	}(file)

	log.Fatal("Some error with the admin key, need key to add links")
	return "" //an empty string represents no admin account allowed
}

func returnString(w io.Writer, s string) {
	_, err := fmt.Fprint(w, s)
	if err != nil {
		log.Println("Error writing string")
		return
	}
}

func correctKey(key string) bool {
	//if key is blank, admin key is blank or the keys don't match then reject
	if key == "" || adminKey == "" || key != adminKey {
		return false
	}
	return true
}

func isInteger(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}
