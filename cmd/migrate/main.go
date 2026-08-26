package main

import (
	"github.com/savioruz/oil/config"
	"github.com/savioruz/oil/migrations"
	"log"
	"os"
)

const (
	argLength = 2
)

func main() {
	if len(os.Args) < argLength {
		log.Fatal("Migration direction (up/down) is required")
	}

	cfg := config.Get()

	switch os.Args[1] {
	case "up":
		if err := migrations.Up(cfg); err != nil {
			log.Fatal(err)
		}
	case "down":
		if err := migrations.Down(cfg); err != nil {
			log.Fatal(err)
		}
	case "drop":
		if err := migrations.Drop(cfg); err != nil {
			log.Fatal(err)
		}
	case "step-up":
		if err := migrations.StepUp(cfg); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Invalid direction. Use 'up', 'down', 'drop' or 'step-up'")
	}
}
