package anidb2json

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

type TitleDB struct {
	XMLName xml.Name `xml:"animetitles" json:"-"`
	Anime   []*Anime `xml:"anime" json:"series"`
}

type Anime struct {
	XMLName     xml.Name `xml:"anime" json:"-"`
	ID          string   `xml:"aid,attr" json:"ID"`
	Titles      []Title  `xml:"title" json:"-"`
	Name        string   `json:"name"`
	Path        []string `json:"paths"`
	Picture     string   `xml:"picture" json:"picture"`
	Description string   `xml:"description" json:"description"`
	Tags        []Tag     `xml:"tags>tag>name" json:"tags"`
	Creators	[]Creator	`xml:"creators>name" json:"creators"`
}

type Title struct {
	XMLName xml.Name `xml:"title"`
	Title   string   `xml:",chardata"`
	Lang    string   `xml:"xml:lang,attr"`
	Type    string   `xml:"type,attr"`
}

type Tag struct {
	XMLName xml.Name `xml:"name" json:"-"`
	Name string `xml:",chardata" json:"name"`
}

type Creator struct {
	XMLName xml.Name `xml:"name" json:"-"`
	Role string `xml:"type,attr" json:"role"`
	Name string `xml:",chardata" json:"name"`
}

type Lookup map[string]*Anime

const infostrip = `Specials|OVA|DVD|BD|Complete|v[0-9]+|(E[pP])*[0-9]+[ ]*[-~][ ]*[0-9]+|[bB]atch|((720)|(1080))([pP]*)|\(.*?\)|\[.*?\]`

const tidystrip = `[ ]+|-|~|:|\?|'|\.|_`

const epnumstrip = `[ ][0-9]{2,}`

const extstrip = `\.(mkv|mp4)$`

func ParseTitleDB(xmldb io.Reader) (tdb *TitleDB, titles Lookup, err error) {
	content, err := ioutil.ReadAll(xmldb)
	if err != nil {
		return
	}
	tdb = &TitleDB{}
	err = xml.Unmarshal(content, tdb)
	if err != nil {
		return
	}
	titles = make(map[string]*Anime)
	for i := range tdb.Anime {
		for _, t := range tdb.Anime[i].Titles {
			if t.Type == "main" {
				tdb.Anime[i].Name = t.Title
			}
			strip := regexp.MustCompile(tidystrip + "|`")
			title := strip.ReplaceAllString(strings.ToLower(t.Title), "")
			titles[title] = tdb.Anime[i]
		}
	}
	return
}

func containsMedia(path string) bool {
	if strings.HasSuffix(path, ".mkv") || strings.HasSuffix(path, ".mp4") {
		return true
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return false
	}
	for _, f := range files {
		if f.IsDir() == false {
			if strings.HasSuffix(f.Name(), ".mkv") || strings.HasSuffix(f.Name(), ".mp4") {
				return true
			}
		}
	}
	return false
}

func cleanName(name string) string {
	if strings.HasSuffix(name, ".mkv") || strings.HasSuffix(name, ".mp4") {
		strip := regexp.MustCompile(extstrip)
		name = strip.ReplaceAllString(name, "")
		strip = regexp.MustCompile(epnumstrip)
		name = strip.ReplaceAllString(name, "")
	}
	strip := regexp.MustCompile(infostrip)
	name = strip.ReplaceAllString(name, "")
	strip = regexp.MustCompile(tidystrip)
	name = strip.ReplaceAllString(name, "")
	return name
}

func (l Lookup) match(fpath string, fi os.FileInfo) (ani *Anime, firstmatch bool, err error) {
	var (
		ok    bool
		files []os.FileInfo
	)
	if ani, ok = l[strings.ToLower(cleanName(fi.Name()))]; ok {
		firstmatch = (len(ani.Path) == 0)
		if fi.IsDir() {
			files, err = ioutil.ReadDir(fpath)
			if err != nil {
				return
			}
			for _, f := range files {
				ani.Path = append(ani.Path, path.Join(fpath, f.Name()))
			}
		} else {
			ani.Path = append(ani.Path, fpath)
		}
		return
	}
	log.Println(fpath, "not matched. Stripped name was:", strings.ToLower(cleanName(fi.Name())))
	err = os.ErrNotExist
	return
}

func (l Lookup) find(fpath string) (found []*Anime) {
	fi, err := ioutil.ReadDir(fpath)
	if err != nil {
		return
	}
	for _, f := range fi {
		abs := path.Join(fpath, f.Name())
		if !containsMedia(abs) {
			continue
		}
		ani, firstmatch, err := l.match(abs, f)
		if firstmatch {
			found = append(found, ani)
		} else if f.IsDir() && err == os.ErrNotExist {
			found = append(found, l.find(abs)...)
		}
	}
	return
}

func (l Lookup) ParseDir(root string) *TitleDB {
	return &TitleDB{Anime: l.find(root)}
}

func fetch(ani *Anime) (content []byte, err error) {
	const baseurl = `http://api.anidb.net:9001/httpapi?request=anime&client=script&clientver=1&protover=1&aid=`
	cl := http.Client{}
	resp, err := cl.Get(baseurl + ani.ID)
	//API specifies 2 second cooldown
	dur, _ := time.ParseDuration("2s")
	time.Sleep(dur)
	if err != nil {
		return
	}
	content, err = ioutil.ReadAll(resp.Body)
	return
}

func FillAdditional(tdb *TitleDB, fpath string) error {
	_, err := os.Stat(fpath)
	if err != nil {
		err = os.Mkdir(fpath, 0777)
	}
	if err != nil {
		return err
	}
	for i := range tdb.Anime {
		cachefilepath := path.Join(fpath, tdb.Anime[i].ID)
		cachefile, err := os.Open(cachefilepath)
		if err != nil {
			b, err := fetch(tdb.Anime[i])
			if err != nil {
				return err
			}
			cachefile, err = os.Create(cachefilepath)
			if err != nil {
				return err
			}
			cachefile.Write(b)
			cachefile.Seek(0, io.SeekStart)
		}
		b, err := ioutil.ReadAll(cachefile)
		if err != nil {
			return err
		}
		err = xml.Unmarshal(b, tdb.Anime[i])
		if err != nil {
			return err
		}
	}
	return nil
}
