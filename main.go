package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
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
//TODO prevent too many shorts from being released on the same day
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

// constant (except for when testing)
var storageJSONFile = "./neatStuff.json"

const shortLinkBase = "https://links.ethohampton.com/"
const numToShowPublic = 5
const adminKeyPath = "./admin.key"

var adminKey string
var port = ":8080"

//These are all "caches" in that they can handle being empty
var itemsToServe []PublicLink
var shortsToServe []Short
var shortsTemplate *template.Template

func main() {
	if len(os.Args) > 1 && isInteger(os.Args[1]) {
		port = ":" + os.Args[1]
	}
	log.Printf("Listening at http://localhost%s", port)

	var err error
	shortsTemplate, err = template.ParseFiles("static/shorts.gohtml")
	if err != nil {
		log.Fatal(err)
	}
	adminKey = getAdminKey(adminKeyPath)

	handlers := getHandlers()

	log.Fatal(http.ListenAndServe(port, handlers))
}

func getHandlers() *http.ServeMux {
	mux := http.ServeMux{}

	mux.HandleFunc("/json", serveLinks)
	mux.HandleFunc("/shorts", serveShorts)
	mux.HandleFunc("/add", serveAdd)
	mux.HandleFunc("/all", serveAll)
	mux.HandleFunc("/", serveInfo)

	return &mux
}

func serveLinks(w http.ResponseWriter, _ *http.Request) {
	if itemsToServe == nil || len(itemsToServe) != numToShowPublic {
		its := loadItemsFromFile(storageJSONFile)
		itemsToServe = getLastNItemsAsPublic(its, numToShowPublic)
	}

	output, _ := json.Marshal(itemsToServe)
	returnString(w, string(output))
	w.Header().Set("Content-Type", " application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveShorts(w http.ResponseWriter, _ *http.Request) {
	//TODO occasionally refresh shorts
	if shortsToServe == nil {
		its := loadItemsFromFile(storageJSONFile)
		shorts := its.Shorts
		for i := len(shorts) - 1; i >= 0; i-- {
			short := shorts[i]
			// make sure shorts aren't being shown before release date
			if time.Now().Before(short.ReleaseDate) {
				shorts = removeShort(shorts, i)
			}
		}
		shortsToServe = shorts
	}

	err := shortsTemplate.Execute(w, struct {
		Shorts []Short
	}{Shorts: shortsToServe})
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		returnString(w, "Couldn't render template")
	}
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
			w.WriteHeader(http.StatusForbidden)
			returnString(w, "Invalid password")
			return
		}

		//for links
		addType := r.FormValue("type")
		switch addType {
		case "link":
			err := addLink(w, r)
			if err != nil {
				return
			}
		case "short":
			err := addShort(w, r)
			if err != nil {
				return
			}
		default:
			w.WriteHeader(400)
			returnString(w, "Invalid Type")
		}

		returnString(w, "Success!")
		log.Println("User added a " + addType)
	} else {
		w.WriteHeader(400)
		returnString(w, "Invalid Method")
	}
}

func addLink(w http.ResponseWriter, r *http.Request) error {
	description := r.FormValue("description")
	url := r.FormValue("url")

	if description == "" || url == "" {
		returnString(w, "Please enter stuff in the form")
		return errors.New("nothing in form")
	}

	//create new item, add to all items and delete currently cached items
	newItem := Link{Description: description, URL: url, AddDate: time.Now()}
	allItems := loadItemsFromFile(storageJSONFile)
	allItems.Links = append(allItems.Links, newItem)
	storeItemsInFile(allItems, storageJSONFile)
	itemsToServe = nil
	return nil
}

func addShort(w http.ResponseWriter, r *http.Request) error {
	title := r.FormValue("title")
	content := r.FormValue("content")
	releaseHours := r.FormValue("releaseHours")

	if title == "" || content == "" || releaseHours == "" {
		returnString(w, "Please enter stuff in the form")
		return errors.New("nothing in form")
	}

	hours, err := strconv.ParseInt(releaseHours, 10, 64)
	if err != nil || hours < 0 {
		returnString(w, "Release hour isn't a positive number")
		return errors.New("release hour not an hour")
	}

	releaseTime := time.Now()
	if hours != 0 {
		releaseTime = releaseTime.Truncate(time.Hour) // to the nearest hour
		releaseTime = releaseTime.Add(time.Hour * time.Duration(hours))
	} else {
		releaseTime = releaseTime.Truncate(time.Second) // if releasing now, then don't need to specify second
	}

	//create new short
	newShort := Short{
		Title:       title,
		Content:     content,
		ID:          generateShortID(),
		ReleaseDate: releaseTime,
		Pinned:      false,
		Kept:        0,
		AddDate:     time.Now(),
	}

	allItems := loadItemsFromFile(storageJSONFile)
	allItems.Shorts = append(allItems.Shorts, newShort)
	storeItemsInFile(allItems, storageJSONFile)
	shortsToServe = nil
	return nil
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
			w.WriteHeader(http.StatusForbidden)
			returnString(w, "Invalid password")
			return
		}

		allItems := loadItemsFromFile(storageJSONFile)
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

func generateShortID() string {
	return RandString(6) // use 6 for now, since 24 possible characters gives us some room
}

// based on Open Location Code Base20 alphabet, add E, L, S and T because I felt like it
const letterBytes = "23456789CEFGHJLMPQRSTVWX"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
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

//removeShort index from shorts array
func removeShort(slice []Short, s int) []Short {
	return append(slice[:s], slice[s+1:]...)
}
