all : bootstrap
.PHONY : all dist install clean

SHELL=/bin/bash

bootstrap : main.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap main.go
	ls -lh bootstrap

myFunction.zip : bootstrap
	zip myFunction.zip bootstrap

dist : myFunction.zip

install : myFunction.zip
	aws lambda update-function-code \
	--function-name tfc-config-upload-event-handler \
	--zip-file fileb://myFunction.zip

clean :
	rm bootstrap myFunction.zip
