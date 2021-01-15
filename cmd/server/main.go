package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/byxorna/homer/internal/version"
	"github.com/byxorna/homer/pkg/server"
)

var (
	y = flag.String("config", "config.yaml", "yaml config for server to load")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: (Package: %s Version: %s Build Date: %s)\n", os.Args[0], version.Package, version.Version, version.BuildDate)
		flag.PrintDefaults()
	}

	flag.Parse()

	buf, err := ioutil.ReadFile(*y)
	if err != nil {
		log.Fatalf("error opening %s: %v", *y, err)
	}
	cfg, err := server.LoadConfig(buf)
	if err != nil {
		log.Fatalf("fuck. cant unmarshal config: %v", err)
	}

	s, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("fuck. unable to instantiate DOH server: %v", err)
	}

	err = s.ListenAndServe()
	if err != nil {
		log.Fatalf("fuck. unable to listen and serve: %v", err)
	}

}
