AWS_ACCOUNT_ID = 253789223556
AWS_REGION = ca-central-1
DOCKER_IMAGE_NAME = run-executor

WATCHED_SOURCES = $(wildcard *.go)

all : run-executor
.PHONY : all dist install clean

SHELL=/bin/bash

run-executor : ${WATCHED_SOURCES} Dockerfile .terraformrc
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o run-executor main.go

dist : run-executor
	docker build --tag tbriot/${DOCKER_IMAGE_NAME}:latest .

test-local : dist
	docker run \
		--name ${DOCKER_IMAGE_NAME} \
		-e AWS_REGION=ca-central-1 \
		-e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
		-e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
		-e AWS_SESSION_TOKEN=${AWS_SESSION_TOKEN} \
		-v shared-volume:/opt/shared-volume \
		-v tfc-cache:/opt/tfc-cache \
		tbriot/${DOCKER_IMAGE_NAME}:latest
	docker container rm ${DOCKER_IMAGE_NAME}

install : dist
	aws ecr get-login-password --region ${AWS_REGION} | sudo docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
	docker build --tag ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest .
	sudo docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest

inspect-volume : dist
	docker run -it \
		--name ${DOCKER_IMAGE_NAME}-inspect-vol \
		-v shared-volume:/opt/shared-volume \
		-v tfc-cache:/opt/tfc-cache \
		tbriot/${DOCKER_IMAGE_NAME}:latest \
		/bin/bash
	docker container rm ${DOCKER_IMAGE_NAME}-inspect-vol

clean :
	rm run-executor || exit 0
	docker container rm ${DOCKER_IMAGE_NAME} || exit 0
	docker container rm ${DOCKER_IMAGE_NAME}-inspect-vol || exit 0
	docker rmi tbriot/${DOCKER_IMAGE_NAME} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME} || exit 0
