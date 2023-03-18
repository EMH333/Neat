package types

import "time"

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
	Pinned      bool      //TODO allow pinning shorts so they don't disappear
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
