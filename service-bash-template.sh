#!/bin/bash
/usr/bin/docker pull {{.Image}}
/usr/bin/docker rm -f {{.Name}}_1
/usr/bin/docker run \
    {{if .Privileged}}--privileged=true {{end}} \
    --restart=always \
    -d \
    --name {{.Name}}_1 \
    {{range .Volumes}}-v {{.}} {{end}} \
    {{range .Environment}}-e {{.}} {{end}} \
    {{range .Ports}}-p {{.}} {{end}} \
    {{.Image}}  {{.Command}}
