AWS_ACCOUNT_ID = 253789223556
AWS_REGION = ca-central-1
DOCKER_IMAGE_NAME = pre-run-helper
SQS_QUEUE_URL = https://sqs.ca-central-1.amazonaws.com/253789223556/tfc-run-events

all : pre-run-helper
.PHONY : all dist install clean publish-msgs

SHELL=/bin/bash

pre-run-helper : main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o pre-run-helper .

dist : pre-run-helper
	docker build --tag tbriot/${DOCKER_IMAGE_NAME}:latest .

test-local: dist
	docker run \
		-e AWS_REGION=ca-central-1 \
		-e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
		-e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
		-e AWS_SESSION_TOKEN=${AWS_SESSION_TOKEN} \
		-v shared-volume:/opt/shared-volume \
		tbriot/${DOCKER_IMAGE_NAME}:latest

install : dist
	aws ecr get-login-password --region ${AWS_REGION} | sudo docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
	docker build --tag ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest .
	sudo docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}:latest

publish-msgs :
	for i in {1..${MSG_NUMBERS}}; \
	do \
		aws sqs send-message --queue-url ${SQS_QUEUE_URL} --message-body file://sqs-msg-payload.json; \
	done; 

clean :
	rm pre-run-helper
	docker rmi tbriot/${DOCKER_IMAGE_NAME} ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${DOCKER_IMAGE_NAME}
