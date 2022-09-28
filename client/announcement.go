package client

import (
	"encoding/json"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/log"
)

func refreshAnnouncement(w *widget.Label, scroll *container.Scroll) {
	announcement, err := getAnnouncement()
	if err != nil || announcement == nil || announcement.Content == "" {
		scroll.Hide()
	} else {
		w.SetText(announcement.Content)
		scroll.Show()
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
	j, err := httpGet(getFullUrl("/announcement"))
	if err != nil {
		return "", err
	}
	return j, nil
}
