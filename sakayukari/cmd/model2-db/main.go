package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/tidwall/buntdb"
	"nyiyui.ca/hato/sakayukari/tal"
)

var dbPath string
var form string
var mode string

func main() {
	flag.StringVar(&dbPath, "db-path", "./model2.test.db", "path to database")
	flag.StringVar(&form, "form", "", "form ID to use")
	flag.StringVar(&mode, "mode", "", "form ID to use")
	flag.Parse()

	if mode != "read" && mode != "write" {
		log.Fatal("mode must be read or write")
	}

	_, err := uuid.Parse(form)
	if err != nil {
		log.Printf("warning: form %s is not a valid UUID: %d", form, err)
	}

	err = main2()
	if err != nil {
		log.Fatal(err)
	}
}

func main2() error {
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return err
	}
	switch mode {
	case "read":
		err = db.View(func(tx *buntdb.Tx) error {
			value, err := tx.Get(fmt.Sprintf("form:%s:data", form))
			if err != nil {
				return err
			}
			var fd tal.FormData
			err = json.Unmarshal([]byte(value), &fd)
			if err != nil {
				log.Fatalf("unmarshalling failed: %s", err)
			}
			log.Printf("found %s", form)
			fmt.Printf("%s", value)
			return nil
		})
		return err
	default:
		panic("not implemented yet")
	}
}
