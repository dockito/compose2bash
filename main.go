package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"text/template"
)

var (
	appName     string
	composePath string
	outputPath  string
)

// Service has the same structure used by docker-compose.yml
type Service struct {
	Name        string
	Image       string
	Ports       []string
	Volumes     []string
	Privileged  bool
	Command     string
	Environment map[string]string
}

// Parses the original Yaml to the Service struct
func loadYaml(filename string) (services map[string]Service, err error) {
	data, err := ioutil.ReadFile(filename)
	if err == nil {
		err = yaml.Unmarshal([]byte(data), &services)
	}
	return
}

// Saves the services data into bash scripts
func saveToBash(services map[string]Service) {
	t, _ := template.ParseFiles("service-bash-template.sh")

	for name, service := range services {
		service.Name = appName + "-" + name

		f, _ := os.Create(path.Join(outputPath, service.Name+".1.sh"))
		defer f.Close()

		t.Execute(f, service)
	}
}

func main() {
	flag.StringVar(&appName, "app", "", "application name")
	flag.StringVar(&composePath, "yml", "docker-compose.yml", "compose file path")
	flag.StringVar(&outputPath, "output", ".", "output directory")

	flag.Parse()

	if appName == "" {
		fmt.Println("Missing app argument")
		os.Exit(1)
	}

	services, err := loadYaml(composePath)
	if err != nil {
		log.Fatalf("error parsing docker-compose.yml %v", err)
	}

	saveToBash(services)

	fmt.Println("Successfully converted Yaml to Bash script.")
}
