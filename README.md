# Nginx Formatter

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) ![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/nginx-formatter) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

Nginx configuration formatter under 10MB size.

## Download

Download the binaries for your system and architecture from the [releases page](https://github.com/soulteary/nginx-formatter/releases).

<img src=".github/dockerhub.png" width="80%" >

If you use docker, you can use the following command ([DockerHub](https://hub.docker.com/r/soulteary/nginx-formatter)):

```bash
docker pull soulteary/nginx-formatter:latest
docker pull soulteary/nginx-formatter:v0.5.0
```

## Usage

Use default parameters to format all configuration files in the current directory:

```bash
./nginx-formatter
```

Use different indentation symbols (You can use spaces, tabs, ` `, `\s`, `\t`) and indentation amounts:

```bash
./nginx-formatter -indent=4 -char=" "
```

Format the configuration file in the specified directory:

```bash
./nginx-formatter -input=./your-dir-path
```

Format a file somewhere and save it in a new directory:

```bash
./nginx-formatter -input=./your-dir-path -output=./your-output-dir
```




## Full parameters supported

List of parameters supported:

```bash
Nginx Formatter

Usage of ./nginx-formatter:
  -char
    	Indent char, defualt:   (default " ")
  -indent int
    	Indent size, defualt: 2 (default 2)
  -input string
    	Input directory
  -output string
    	Output directory
```

## Credits

Formatter Components

- soulteary Modify the JavaScript version for golang execution, under [Apache-2.0 license], 18/04/2023:
  - simplify the program, fix bugs, improve running speed, and allow running in golang
  - https://github.com/soulteary/nginx-formatter
- Yosef Ported the JavaScript beautifier under [Apache-2.0 license], 24/08/2016
  - https://github.com/vasilevich/nginxbeautifier
- Slomkowski Created a beautifier for nginx config files with Python under [Apache-2.0 license], 24/06/2016
  - https://github.com/1connect/nginx-config-formatter (https://github.com/slomkowski/nginx-config-formatter)

Runtime dependent components

- ECMAScript 5.1(+) implementation in Go
  - https://github.com/dop251/goja
