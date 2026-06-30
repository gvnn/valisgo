#!/usr/bin/env bash
set -e

REGISTRY_URL="http://localhost:8080/registries/myfile/repositories/myrepo"

echo "Uploading test file..."
echo "Hello, Valisgo File Registry!" > test-upload.txt
curl -f -v -u dummy:dummy --upload-file test-upload.txt "${REGISTRY_URL}/test-upload.txt"
echo -e "\nUpload successful."

echo "Downloading test file..."
curl -f -s -u dummy:dummy -o downloaded.txt "${REGISTRY_URL}/test-upload.txt"

if ! grep -q "Hello, Valisgo File Registry!" downloaded.txt; then
    echo "Downloaded file content mismatch!"
    cat downloaded.txt
    exit 1
fi

echo "File integration test completed successfully!"
rm test-upload.txt downloaded.txt
