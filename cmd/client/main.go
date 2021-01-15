package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/byxorna/homer/internal/version"
	"github.com/byxorna/homer/pkg/client"
)

var (
	yamlConfig = flag.String("config", "config.yaml", "yaml config for client to load")
)

func main() {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: (Package: %s Version: %s Build Date: %s)\n", os.Args[0], version.Package, version.Version, version.BuildDate)
		flag.PrintDefaults()
	}

	flag.Parse()

	buf, err := ioutil.ReadFile(*yamlConfig)
	if err != nil {
		log.Fatalf("error opening %s: %v", *yamlConfig, err)
	}
	cfg, err := client.LoadConfig(buf)
	if err != nil {
		log.Fatalf("fuck. cant unmarshal config: %v", err)
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("fuck. unable to instantiate client: %v", err)
	}

	err = c.ListenAndServe()
	if err != nil {
		log.Fatalf("fuck. unable to listen and serve: %v", err)
	}
}
