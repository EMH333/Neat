package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"ethohampton.com/Neat/internal/types"
	"ethohampton.com/Neat/internal/util"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

//TODO before shorts release:
// - add SEO to shorts page
// - prep brief blog post
// - preload a whole bunch of shorts
// - add link from main website

//TODO after release:
// - page per short
// - paginate shorts

// constant (except for when testing)
var storageJSONFile = "./neatStuff.json"
var storageMutex sync.RWMutex

const shortLinkBase = "https://links.ethohampton.com/"
const numLinksToShowPublic = 5
const numShortsToShowPublic = 10
const adminKeyPath = "./admin.key"

var adminKey string
var port = ":8080"

//These are all "caches" in that they can handle being empty
var itemsToServe []types.PublicLink
var shortsToServe []types.Short
var shortsTemplate *template.Template
var shortsLastUpdated time.Time

func main() {
	if len(os.Args) > 1 && util.IsInteger(os.Args[1]) {
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
	if itemsToServe == nil || len(itemsToServe) != numLinksToShowPublic {
		// make sure we have a read lock before continuing
		storageMutex.RLock()
		defer storageMutex.RUnlock()

		its := util.LoadItemsFromFile(storageJSONFile)
		itemsToServe = getLastNItemsAsPublic(its, numLinksToShowPublic)
	}

	output, _ := json.Marshal(itemsToServe)
	returnString(w, string(output))
	w.Header().Set("Content-Type", " application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func serveShorts(w http.ResponseWriter, _ *http.Request) {
	//make sure shorts are correct before serving them
	updateShortsToServe()

	err := shortsTemplate.Execute(w, struct {
		Shorts []types.Short
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

		http.ServeFile(w, r, "./static/addSuccess.html")
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

	// make sure clear to write
	storageMutex.Lock()
	defer storageMutex.Unlock()

	//create new item, add to all items and delete currently cached items
	newItem := types.Link{Description: description, URL: url, AddDate: time.Now().Truncate(time.Millisecond)}
	allItems := util.LoadItemsFromFile(storageJSONFile)
	allItems.Links = append(allItems.Links, newItem)
	util.StoreItemsInFile(allItems, storageJSONFile)
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

	// find release time to make sure there is separation
	tmpShorts := util.LoadItemsFromFile(storageJSONFile).Shorts
	releaseTime = util.FindNextValidShortReleaseTime(releaseTime, tmpShorts)

	//create new short
	newShort := types.Short{
		Title:       title,
		Content:     content,
		ID:          generateShortID(),
		ReleaseDate: releaseTime,
		Pinned:      false,
		Kept:        0,
		AddDate:     time.Now().Truncate(time.Millisecond), // don't need this much precision
	}

	// make sure clear to write
	storageMutex.Lock()
	defer storageMutex.Unlock()

	allItems := util.LoadItemsFromFile(storageJSONFile)
	allItems.Shorts = append(allItems.Shorts, newShort)
	util.StoreItemsInFile(allItems, storageJSONFile)
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

		allItems := util.LoadItemsFromFile(storageJSONFile)
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

func getLastNItemsAsPublic(items *types.ContentStorage, n int) []types.PublicLink {
	length := len(items.Links)
	if length < 0 {
		return make([]types.PublicLink, 0)
	}

	if length < n {
		n = length
	}

	transform := items.Links[length-n:] // grab last n elements of items we are dealing with
	out := make([]types.PublicLink, n)

	//for each item, set the right URL (short or not) and description
	for i := 0; i < n; i++ {
		var newI types.PublicLink
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

func updateShortsToServe() {
	// refresh shorts every hour in case something has changed
	if shortsLastUpdated.Add(time.Hour).Before(time.Now()) {
		shortsToServe = nil
	}

	if shortsToServe == nil {
		// make sure we have a read lock before continuing
		storageMutex.RLock()
		defer storageMutex.RUnlock()

		its := util.LoadItemsFromFile(storageJSONFile)
		shorts := its.Shorts
		for i := len(shorts) - 1; i >= 0; i-- {
			short := shorts[i]
			// make sure shorts aren't being shown before release date
			if time.Now().Before(short.ReleaseDate) {
				shorts = removeShort(shorts, i)
			}
		}
		//sort so they are displayed with the most recently released ones first
		sort.Slice(shorts[:], func(i, j int) bool {
			return shorts[i].ReleaseDate.After(shorts[j].ReleaseDate)
		})

		//limit to numShortsToShowPublic shorts
		if len(shorts) > numShortsToShowPublic {
			shorts = shorts[:numShortsToShowPublic]
		}

		shortsToServe = shorts
		shortsLastUpdated = time.Now()
	}
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
	return util.RandString(6) // use 6 for now, since 24 possible characters gives us some room
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

//removeShort index from shorts array
func removeShort(slice []types.Short, s int) []types.Short {
	return append(slice[:s], slice[s+1:]...)
}
