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
	--function-name tfc-create-configuration-versions \
	--zip-file fileb://myFunction.zip

test:
	./get_presigned_url.sh

clean :
	rm bootstrap myFunction.zip

