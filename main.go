package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

var (
	appName         string
	composePath     string
	outputPath      string
	dockerHostConn  string
	interactiveBash bool
)

const bashTemplate = `#!/bin/bash
/usr/bin/docker {{.DockerHostConnCmdArg}} pull {{.Service.Image}}

if /usr/bin/docker {{.DockerHostConnCmdArg}} ps -a | grep --quiet {{.Service.Name}}_1 ; then
	/usr/bin/docker {{.DockerHostConnCmdArg}} rm -f {{.Service.Name}}_1
fi

{{if .InteractiveBash}}
while [ "$#" -gt 0 ]; do case "$1" in
    --interactive-bash) interactivebash="true"; shift 1;;
    *) shift;;
  esac
done

if [[ $interactivebash == "true" ]]; then
	/usr/bin/docker {{.DockerHostConnCmdArg}} run \
		{{if .Service.Privileged}}--privileged=true {{end}} \
		-ti \
		--name {{.Service.Name}}_1 \
		{{if .Service.HostName}}--hostname={{.Service.HostName}} {{end}} \
		{{if .Service.Net}}--net={{.Service.Net}} {{end}} \
		{{range .Service.Volumes}}-v {{.}} {{end}} \
		{{range .Service.Links}}--link {{.}} {{end}} \
		{{range $key, $value := .Service.Environment}}-e {{$key}}="{{$value}}" {{end}} \
		{{range .Service.Ports}}-p {{.}} {{end}} \
		{{range .Service.Env_File}}--env-file {{.}} {{end}} \
                {{if .Service.Log_Driver}}--log-driver {{.Service.Log_Driver}} {{end}} \
                {{range $key, $value := .Service.Log_Opt}}--log-opt {{$key}}={{$value}} {{end}} \
		{{.Service.Image}} bash
else
	/usr/bin/docker {{.DockerHostConnCmdArg}} run \
		{{if .Service.Privileged}}--privileged=true {{end}} \
		--restart=always \
		-d \
		--name {{.Service.Name}}_1 \
		{{if .Service.HostName}}--hostname={{.Service.HostName}} {{end}} \
		{{if .Service.Net}}--net={{.Service.Net}} {{end}} \
		{{range .Service.Volumes}}-v {{.}} {{end}} \
		{{range .Service.Links}}--link {{.}} {{end}} \
		{{range $key, $value := .Service.Environment}}-e {{$key}}="{{$value}}" {{end}} \
		{{range .Service.Ports}}-p {{.}} {{end}} \
		{{range .Service.Env_File}}--env-file {{.}} {{end}} \
                {{if .Service.Log_Driver}}--log-driver {{.Service.Log_Driver}} {{end}} \
                {{range $key, $value := .Service.Log_Opt}}--log-opt {{$key}}={{$value}} {{end}} \
		{{.Service.Image}} {{.Service.Command}}
fi
{{else}}
/usr/bin/docker {{.DockerHostConnCmdArg}} run \
	{{if .Service.Privileged}}--privileged=true {{end}} \
	--restart=always \
	-d \
	--name {{.Service.Name}}_1 \
	{{if .Service.HostName}}--hostname={{.Service.HostName}} {{end}} \
	{{if .Service.Net}}--net={{.Service.Net}} {{end}} \
	{{range .Service.Volumes}}-v {{.}} {{end}} \
	{{range .Service.Links}}--link {{.}} {{end}} \
	{{range $key, $value := .Service.Environment}}-e {{$key}}="{{$value}}" {{end}} \
	{{range .Service.Ports}}-p {{.}} {{end}} \
	{{range .Service.Env_File}}--env-file {{.}} {{end}} \
        {{if .Service.Log_Driver}}--log-driver {{.Service.Log_Driver}} {{end}} \
        {{range $key, $value := .Service.Log_Opt}}--log-opt {{$key}}={{$value}} {{end}} \
	{{.Service.Image}} {{.Service.Command}}
{{end}}
`

// ScriptDataTemplate contains the whole data configuration used to fill the script
type ScriptDataTemplate struct {
	AppName              string
	DockerHostConnCmdArg string
	InteractiveBash      bool
	Service              Service
}

// Service has the same structure used by docker-compose.yml
type Service struct {
	Name        string
	HostName    string
	Image       string
	Net         string
	Ports       []string
	Volumes     []string
	Env_File    []string
	Links       []string
	Privileged  bool
	Command     string
	Environment map[string]string
        Log_Driver   string
        Log_Opt      map[string]string
}

// Parses the original Yaml to the Service struct
func loadYaml(filename string) (services map[string]Service, err error) {
	data, err := ioutil.ReadFile(filename)
	if err == nil {
		err = yaml.Unmarshal([]byte(data), &services)
	}
	return
}

func setLinksWithAppName(service *Service) {
	for i := range service.Links {
		links := strings.Split(service.Links[i], ":")
		containerName := links[0]
		containerAlias := containerName + "_1"
		if len(links) > 1 {
			containerAlias = links[1]
		}

		service.Links[i] = fmt.Sprintf("%s-%s_1:%s", appName, containerName, containerAlias)
	}
}

func removeBlankLinkes(path string) {
	re := regexp.MustCompile(`(?m)^\s*\\\n`)
	read, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	newContents := re.ReplaceAllString(string(read), "")

	err = ioutil.WriteFile(path, []byte(newContents), 0)
	if err != nil {
		panic(err)
	}
}

func buildScriptDataTemplate(serviceName string, service Service) ScriptDataTemplate {
	// common data template for all services from the same app
	data := ScriptDataTemplate{
		AppName:         appName,
		InteractiveBash: interactiveBash,
	}

	if dockerHostConn != "" {
		data.DockerHostConnCmdArg = "--host=" + dockerHostConn
	}

	// specific data for each service
	service.Name = appName + "-" + serviceName
	setLinksWithAppName(&service)
	data.Service = service

	return data
}

// Saves the services data into bash scripts
func saveToBash(services map[string]Service) (err error) {
	t := template.New("service-bash-template")
	t, err = t.Parse(bashTemplate)
	if err != nil {
		return err
	}

	for name, service := range services {
		data := buildScriptDataTemplate(name, service)
		fileName := path.Join(outputPath, data.Service.Name+".1.sh")

		f, _ := os.Create(fileName)
		defer f.Close()
		t.Execute(f, data)

		removeBlankLinkes(fileName)
	}

	return nil
}

func main() {
	version := flag.Bool("v", false, "show the current version")
	flag.StringVar(&appName, "app", "", "application name")
	flag.StringVar(&composePath, "yml", "docker-compose.yml", "compose file path")
	flag.StringVar(&outputPath, "output", ".", "output directory")
	flag.StringVar(&dockerHostConn, "docker-host", "", "docker host connection")
	flag.BoolVar(&interactiveBash, "interactive-bash", false, "include option to run the generated script with interactive bash")

	flag.Parse()

	if *version {
		fmt.Println("compose2bash version 1.5.0")
		return
	}

	if appName == "" {
		fmt.Println("Missing app argument")
		os.Exit(1)
	}

	services, err := loadYaml(composePath)
	if err != nil {
		log.Fatalf("error parsing docker-compose.yml %v", err)
	}

	if err = saveToBash(services); err != nil {
		log.Fatalf("error saving bash template %v", err)
	}

	fmt.Println("Successfully converted Yaml to Bash script.")
}
