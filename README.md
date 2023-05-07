# Nginx Formatter

[![CodeQL](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/github-code-scanning/codeql) [![Codecov](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/codecov.yml) [![Security Scan](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/scan.yml) [![Release](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml/badge.svg)](https://github.com/soulteary/nginx-formatter/actions/workflows/release.yaml) ![Go Report Card](https://goreportcard.com/badge/github.com/soulteary/nginx-formatter) [![Docker Image](https://img.shields.io/docker/pulls/soulteary/nginx-formatter.svg)](https://hub.docker.com/r/soulteary/nginx-formatter)

<p style="text-align: center;">
  <a href="README.md" target="_blank">ENGLISH</a> | <a href="README_CN.md">中文文档</a>
</p>

<img src=".github/logo.png" width="120" >

Nginx configuration formatter ~10MB size, support CLI, WebUI, x86, ARM, Linux, macOS.

<img src=".github/preview.png">

## Download

Download the binaries for your system and architecture from the [releases page](https://github.com/soulteary/nginx-formatter/releases).

<img src=".github/dockerhub.png" width="80%" >

If you use docker, you can use the following command ([DockerHub](https://hub.docker.com/r/soulteary/nginx-formatter)):

```bash
docker pull soulteary/nginx-formatter:latest
docker pull soulteary/nginx-formatter:v1.1.1
```

## Usage

Use default parameters to format all configuration files in the current directory:

```bash
./nginx-formatter
```

### Common Usage (CLI & WebUI)

Use different indentation symbols (You can use spaces, tabs, ` `, `\s`, `\t`) and indentation amounts:

```bash
./nginx-formatter -indent=4 -char=" "
```

### CLI Usage

Format the configuration file in the specified directory:

```bash
./nginx-formatter -input=./your-dir-path
```

Format a file somewhere and save it in a new directory:

```bash
./nginx-formatter -input=./your-dir-path -output=./your-output-dir
```

### WebUI Usage

Start the web interface:

```bash
./nginx-formatter -web
```

specified the port:

```bash
./nginx-formatter -web -port=8123
```

### Docker Usage

There is no difference between using parameters in Docker and the above, for example, we start a Web UI formatting tool service in Docker:

```bash
docker run --rm -it -p 8080:8080 soulteary/nginx-formatter:v1.1.1 -web
```

If you want to format the configuration of the current directory, you can use the program in Docker with a command similar to the following:

```bash
docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter:v1.1.1 -input=/app
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
  -port 8080
    	WebUI Port, defualt: 8080 (default 8080)
  -web false
    	Enable WebUI, defualt: false
```

## Credits

Formatter Components

- Slomkowski Created a beautifier for nginx config files with Python under [Apache-2.0 license], 24/06/2016
  - https://github.com/1connect/nginx-config-formatter (https://github.com/slomkowski/nginx-config-formatter)
- Yosef Ported the JavaScript beautifier under [Apache-2.0 license], 24/08/2016
  - https://github.com/vasilevich/nginxbeautifier
- soulteary Modify the JavaScript version for golang execution, under [Apache-2.0 license], 18/04/2023:
  - simplify the program, fix bugs, improve running speed, and allow running in golang
  - https://github.com/soulteary/nginx-formatter

Runtime dependent Components

- ECMAScript 5.1(+) implementation in Go, under [MIT license].
  - https://github.com/dop251/goja

Web Components

- Gin is a HTTP web framework written in Go (Golang), under [MIT license].
  - https://github.com/gin-gonic/gin
- Code Mirror, in-browser code editor, under [MIT license].
  - https://github.com/codemirror/codemirror5
