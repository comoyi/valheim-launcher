package client

import (
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/log"
	"net/url"
)

var announcementContent string
var announcementHash = ""

func refreshAnnouncement(w *widget.Label, box *fyne.Container) {
	announcement, err := getAnnouncement()
	if err != nil || announcement == nil {
		box.Hide()
		return
	}
	content := ""
	if announcement.Content != "" {
		content = announcement.Content
		announcementHash = announcement.Hash
	} else {
		if announcement.Hash != "" {
			if announcementHash == announcement.Hash {
				content = announcementContent
			}
		}
	}

	announcementContent = content

	if content == "" {
		box.Hide()
	} else {
		w.SetText(content)
		box.Show()
	}
}

func getAnnouncement() (*Announcement, error) {
	j, err := fetchAnnouncement()
	if err != nil {
		log.Debugf("request failed, err: %v\n", err)
		return nil, err
	}
	var announcement *Announcement
	err = json.Unmarshal([]byte(j), &announcement)
	if err != nil {
		log.Debugf("json.Unmarshal failed, err: %v\n", err)
		return nil, err
	}
	return announcement, nil
}

func fetchAnnouncement() (string, error) {
	finalUrl := ""
	if announcementContent != "" {
		q := url.Values{}
		q.Set("hash", announcementHash)
		finalUrl = fmt.Sprintf("%s%s", getFullUrl("/announcement"), "?"+q.Encode())
	} else {
		finalUrl = getFullUrl("/announcement")
	}
	j, err := httpGet(finalUrl)
	if err != nil {
		return "", err
	}
	return j, nil
}
