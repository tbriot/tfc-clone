#!/bin/bash

echo "--------------------------------------------------------------------------------"
echo "Get presigned URL"

PRESIGNED_URL=$(
    curl \
    --request POST \
    --silent \
    --header "Content-Type: application/json" \
    --data @request_payload.json \
    https://a2fv2x3429.execute-api.ca-central-1.amazonaws.com/test/workspaces/1/configuration-versions \
    | jq --raw-output '.data.attributes."upload-url"'
)

echo "--------------------------------------------------------------------------------"
echo "Presigned URL=${PRESIGNED_URL}"

echo "--------------------------------------------------------------------------------"
echo "Uploading file"
echo "--------------------------------------------------------------------------------"
curl -I --request PUT --upload-file ./tf-config.tar.gz ${PRESIGNED_URL}
