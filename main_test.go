package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func writeFile(fileName string, data []byte) {
	f, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
			return
		}
	}(f)

	_, err = f.Write(data)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func TestLinkHTTP(t *testing.T) {
	// create a basic content file
	file := ContentStorage{
		Links: []Link{
			{
				URL:         "testing1",
				Description: "IDK",
			},
			{
				URL:         "testing2",
				Description: "IDK",
			},
			{
				URL:         "testing3",
				Description: "IDK",
			},
			{
				URL:         "testing4",
				Description: "IDK",
			},
			{
				URL:         "testing5",
				Description: "IDK",
			},
		},
		Shorts:                  nil,
		ShortVisibilityDuration: 0,
	}
	dir, err := os.MkdirTemp("", "Neat")
	if err != nil {
		t.Fatal(err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(dir)
	fileLoc := dir + "/testing.json"

	j, _ := json.Marshal(file)
	writeFile(fileLoc, j)

	// set needed variables
	storageJSONFile = fileLoc
	handlers := getHandlers()
	adminKey = "testing"

	// test getting all
	resp := httptest.NewRecorder()
	body := strings.NewReader("password=testing&url=testing6&description=somethingNew&type=link")
	req, err := http.NewRequest("POST", "/add", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//check status code
		if resp.Code != http.StatusOK {
			t.Fatalf("Wrong status %d", resp.Code)
		}
	}

	// test JSON
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/json", nil)
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if p, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//make sure we can parse it back into a proper json array, and it has the right items
		var items []PublicLink
		err := json.Unmarshal(p, &items)
		if err != nil {
			t.Fatal(err)
		}

		if len(items) != 5 {
			t.Fatalf("returned array had %d items, expected 5", len(items))
		}

		for _, item := range items {
			if item.URL == "testing1" {
				t.Fatal("returned array had 'testing1' item")
			}
		}
	}

	// test getting all
	resp = httptest.NewRecorder()
	body = strings.NewReader("password=testing")
	req, err = http.NewRequest("POST", "/all", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if p, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//make sure we can parse it back into a proper json array, and it has the right items
		var items ContentStorage
		err := json.Unmarshal(p, &items)
		if err != nil {
			t.Fatal(err, string(p))
		}

		if len(items.Links) != 6 {
			t.Fatalf("returned array had %d items, expected 6", len(items.Links))
		}
	}

	// test getting all with wrong password
	resp = httptest.NewRecorder()
	body = strings.NewReader("password=wrong")
	req, err = http.NewRequest("POST", "/all", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//check status code
		if resp.Code != http.StatusForbidden {
			t.Fatalf("Wrong status %d", resp.Code)
		}
	}

	// test adding with wrong password
	resp = httptest.NewRecorder()
	body = strings.NewReader("password=wrong")
	req, err = http.NewRequest("POST", "/add", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//check status code
		if resp.Code != http.StatusForbidden {
			t.Fatalf("Wrong status %d", resp.Code)
		}
	}

	// test random page (not found)
	resp = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/random-does-not-exist", nil)
	if err != nil {
		t.Fatal(err)
	}

	handlers.ServeHTTP(resp, req)
	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		t.Fail()
	} else {
		//check status code
		if resp.Code != http.StatusNotFound {
			t.Fatalf("Wrong status %d", resp.Code)
		}
	}
}

func Test_correctKey(t *testing.T) {
	//make sure to set admin key
	adminKey = "testing"

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "correct", key: "testing", want: true},
		{name: "incorrect", key: "testingWrong", want: false},
		{name: "blank", key: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := correctKey(tt.key); got != tt.want {
				t.Errorf("correctKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getLastNItemsAsPublic(t *testing.T) {
	type args struct {
		items *ContentStorage
		n     int
	}
	tests := []struct {
		name string
		args args
		want []PublicLink
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLastNItemsAsPublic(tt.args.items, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLastNItemsAsPublic() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isInteger(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "yes",
			s:    "123",
			want: true,
		},
		{
			name: "no",
			s:    "nope",
			want: false,
		},
		{
			name: "maybe so",
			s:    "n123",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isInteger(tt.s); got != tt.want {
				t.Errorf("isInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadItemsFromFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want *ContentStorage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := loadItemsFromFile(tt.args.filename); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadItemsFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_storeItemsInFile(t *testing.T) {
	type args struct {
		items    *ContentStorage
		filename string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeItemsInFile(tt.args.items, tt.args.filename)
		})
	}
}
