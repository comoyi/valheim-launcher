package client

var versionText = "1.0.2"

//var versionNo = 101
//
//var haveNewVersion = false
//var lastVersionInfo *Version
//
//type Version struct {
//	Version     string `json:"version"`
//	VersionNo   int    `json:"version_no"`
//	Description string `json:"description"`
//	ChangeLog   string `json:"change_log"`
//}
//
//func CheckNewVersion() {
//	if haveNewVersion {
//		return
//	}
//
//	doCheckNewVersion()
//}
//
//func doCheckNewVersion() {
//	v, err := getLastVersionInfo()
//	if err != nil {
//		return
//	}
//
//	lastVersionInfo = v
//
//	if versionNo < v.VersionNo {
//		haveNewVersion = true
//	}
//}
//
//func getLastVersionInfo() (*Version, error) {
//	// version check url
//	url := ""
//	d, err := httpGet(url)
//	if err != nil {
//		return nil, err
//	}
//	var v *Version
//	err = json.Unmarshal([]byte(d), &v)
//	if err != nil {
//		return nil, err
//	}
//	return v, nil
//}
