package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/majiru/anidb2json"
)

func main() {
	var mediadir, cachedir, titledb string
	if len(os.Args) < 3 {
		fmt.Printf("Useage: %s titledb mediadir [cachedir]\n", os.Args[0])
		os.Exit(1)
	}
	if len(os.Args) >= 4 {
		cachedir = os.Args[3]
	} else {
		cachedir = "./cache"
	}
	titledb = os.Args[1]
	mediadir = os.Args[2]

	f, err := os.Open(titledb)
	if err != nil {
		log.Fatal(err)
	}
	tdb, titles, err := anidb2json.ParseTitleDB(f)
	if err != nil {
		log.Fatal(err)
	}
	tdb, err = anidb2json.Parsedir(mediadir, titles)
	if err != nil {
		log.Fatal(err)
	}
	err = anidb2json.FillAdditional(tdb, cachedir)
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.Marshal(tdb)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(b)
}