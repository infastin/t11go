package main

import (
	"log"

	"github.com/infastin/t11go/internal/app"
)

func main() {
	app, err := app.NewApplication()
	if err != nil {
		log.Fatal(err)
	}

	err = app.Watch()
	if err != nil {
		log.Fatal(err)
	}

	app.Run()
}
