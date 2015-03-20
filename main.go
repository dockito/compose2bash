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

// Service ...
type Service struct {
	Name         string
	Image        string
	Ports        []string
	Volumes      []string
	Privileged   bool
	Command      string
	Links        []string
	Environments []string
}

func convertFigToBash(appName string, serviceName interface{}, serviceConfig interface{}) Service {
	service := Service{}

	name, _ := serviceName.(string)
	service.Name = appName + "-" + name

	serviceConfig1, _ := serviceConfig.(map[interface{}]interface{})

	service.Image, _ = serviceConfig1["image"].(string)

	service.Privileged = (serviceConfig1["privileged"] != nil)

	if command := serviceConfig1["command"]; command != nil {
		service.Command = command.(string)
	}

	ports, _ := serviceConfig1["ports"].([]interface{})
	if ports != nil {
		for _, port := range ports {
			p, _ := port.(string)
			service.Ports = append(service.Ports, p)
		}
	}

	volumes, _ := serviceConfig1["volumes"].([]interface{})
	if volumes != nil {
		for _, volume := range volumes {
			p, _ := volume.(string)
			service.Volumes = append(service.Volumes, p)
		}
	}

	environment, _ := serviceConfig1["environment"].(map[interface{}]interface{})
	if environment != nil {
		for _, env := range environment {
			p, _ := env.(string)
			service.Ports = append(service.Ports, p)
		}
	}

	return service
}

func loadYaml(filename string) (map[interface{}]interface{}, error) {
	m := make(map[interface{}]interface{})

	data, err := ioutil.ReadFile(filename)

	if err == nil {
		err = yaml.Unmarshal([]byte(data), &m)
	}

	return m, err
}

func main() {
	appName := flag.String("app", "", "application name")
	composePath := flag.String("yml", "docker-compose.yml", "compose file path")
	outputPath := flag.String("output", ".", "output directory")

	flag.Parse()

	if *appName == "" {
		fmt.Println("Missing app argument")
		os.Exit(1)
	}

	data, err := loadYaml(*composePath)
	if err != nil {
		log.Fatalf("error parsing docker-compose.yml %v", err)
	}

	services := make(map[string]Service)

	for serviceName, serviceConfig := range data {
		service := convertFigToBash(*appName, serviceName, serviceConfig)
		services[service.Name] = service
	}

	t, _ := template.ParseFiles("service-bash-template.sh")

	for _, service := range services {
		f, _ := os.Create(path.Join(*outputPath, service.Name+".1.sh"))
		defer f.Close()

		t.Execute(f, service)
	}
}
