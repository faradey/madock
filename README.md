# madock
Local development environment based on Docker for Magento

## Description
`madock` is a local Docker-based environment that allows you to run Magento2 (also Magento1) projects.
This project is written in Golang and is distributed under a MIT License.

## Key Features
* Automatic project setup
* Two or more projects can work simultaneously
* Cron support
* Flexible configuration for each project
* Database import and export in two clicks
* Simple viewing of logs with one command
* Debug support

## Tested on
* Linux (Ubuntu 20.04)
* macOS (Monterey)

## Installation

You need 5 things on your local machine: `git`, `docker`, `docker-compose`, `golang` and `madock`

Follow the installation steps for your system.
<details>
<summary>Mac</summary>

1. Install [Docker](https://docs.docker.com/docker-for-mac/install/)
2. Install [Golang](https://go.dev/doc/install)
3. Install [Mutagen](https://mutagen.io/documentation/introduction/installation)
4. Clone this repo and follow into folder "madock"
```
git clone git@github.com:faradey/madock.git
```
5. Compile
```
Run command below for Apple M1

GOARCH=arm64 go build -o madock
```
```
Run command below for Apple Intel

go build -o madock
```
6. Add `madock` bin into your `$PATH`
```
ln -s absolute_path_to_your_madok_dir/madock /usr/local/bin/
```
7. Open a new terminal tab/window and check that `madock` works
```
which madock
madock
```
8. Optionally you can also apply these performance tweaks
    * [http://markshust.com/2018/01/30/performance-tuning-docker-mac](http://markshust.com/2018/01/30/performance-tuning-docker-mac)
</details>

<details>
<summary>Linux</summary>

1. Install docker
   * Install Docker on [Debian](https://docs.docker.com/engine/installation/linux/docker-ce/debian/)
   * Install Docker on [Ubuntu](https://docs.docker.com/engine/installation/linux/docker-ce/ubuntu/)
   * Install Docker on [CentOS](https://docs.docker.com/engine/installation/linux/docker-ce/centos/)
2. Install [Golang](https://go.dev/doc/install)
3. Clone this repo and follow into folder "madock"
```
git clone git@github.com:faradey/madock.git
```
4. Compile
```
go build -o madock
```
5. Add `madock` bin into your `$PATH`
```
ln -s absolute_path_to_your_madok_dir/madock /usr/local/bin/
```
6. Open a new terminal tab/window and check that `madock` works
```
which madock
madock
```
</details>

## Project Setup
