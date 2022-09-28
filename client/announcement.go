package client

import (
	"encoding/json"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/comoyi/valheim-launcher/log"
)

func refreshAnnouncement(w *widget.Label, box *fyne.Container) {
	announcement, err := getAnnouncement()
	if err != nil || announcement == nil || announcement.Content == "" {
		box.Hide()
	} else {
		w.SetText(announcement.Content)
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
	j, err := httpGet(getFullUrl("/announcement"))
	if err != nil {
		return "", err
	}
	return j, nil
}
