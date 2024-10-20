package main

import (
	"github.com/RealFax/packaged"
	"github.com/RealFax/packaged/example/1-multiple/conf"
	"github.com/RealFax/packaged/example/1-multiple/server"
	"log"
)

func main() {
	packaged.Register(conf.NewEntry, packaged.WithSetup())
	packaged.Register(server.NewEntry)

	if err := packaged.Run(); err != nil {
		log.Fatal(err)
	}

	packaged.Wait()
}
