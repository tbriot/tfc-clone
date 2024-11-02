AWS_ACCOUNT_ID = 253789223556
AWS_REGION = ca-central-1
DOCKER_IMAGE_NAME = run-executor

WATCHED_SOURCES = $(wildcard *.go)

all : run-executor
.PHONY : all dist install clean

SHELL=/bin/bash

run-executor : ${WATCHED_SOURCES}
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o run-executor main.go

dist : run-executor
	docker build --tag tbriot/${DOCKER_IMAGE_NAME}:latest .

install : dist
	aws ecr get-login-password --region ${AWS_REGION} | sudo docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
	docker build --tag ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest .
	sudo docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest

clean :
	rm run-executor
	docker rmi tbriot/${DOCKER_IMAGE_NAME} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}
