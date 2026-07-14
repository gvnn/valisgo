#!/usr/bin/env bash
set -e

# Change to project root
cd "$(dirname "$0")/../../.."
PROJECT_ROOT=$(pwd)

# Wait for server to start if it's not running
echo "Checking if server is running..."
if ! curl -s http://localhost:8080 > /dev/null; then
  echo "Server is not running. Please start the server first."
  exit 1
fi

PROXY_URL="http://localhost:8080/registries/mygo/repositories/go-proxy"
LOCAL_URL="http://localhost:8080/registries/mygo/repositories/go-local"
VIRTUAL_URL="http://localhost:8080/registries/mygo/repositories/go-virtual"

echo "Publishing dummy package..."

MODULE="example.com/my-dummy-pkg"
VERSION="v1.0.2"
PREFIX="${MODULE}@${VERSION}"

# Package it up in a temporary workspace
WORKDIR=$(mktemp -d)
cd "$WORKDIR"

# Create zip using our custom pack.go script that uses golang.org/x/mod/zip
(cd "$PROJECT_ROOT" && go run "$PROJECT_ROOT/tests/integration/golang/pack.go" "$PREFIX" "$PROJECT_ROOT/tests/integration/golang/dummy-pkg" "$WORKDIR/${VERSION}.zip")

# Create .info
cat <<EOF > "${VERSION}.info"
{"Version":"$VERSION","Time":"2023-01-01T00:00:00Z"}
EOF

# Copy go.mod to version.mod
cp "$PROJECT_ROOT/tests/integration/golang/dummy-pkg/go.mod" "${VERSION}.mod"

# Upload to local repo
echo "Uploading ${VERSION}.info"
curl -s -X PUT --data-binary @"${VERSION}.info" "$LOCAL_URL/${MODULE}/@v/${VERSION}.info"
echo "Uploading ${VERSION}.mod"
curl -s -X PUT --data-binary @"${VERSION}.mod" "$LOCAL_URL/${MODULE}/@v/${VERSION}.mod"
echo "Uploading ${VERSION}.zip"
curl -s -X PUT --data-binary @"${VERSION}.zip" "$LOCAL_URL/${MODULE}/@v/${VERSION}.zip"

echo "Testing Go proxy fetching..."
mkdir -p "$PROJECT_ROOT/tests/integration/golang/workspace"
cd "$PROJECT_ROOT/tests/integration/golang/workspace"

# Cleanup workspace
rm -f go.mod go.sum main.go testapp

go mod init test-project

# Tell Go to use our virtual registry
export GOPROXY="$VIRTUAL_URL,direct"
export GOSUMDB=off

# 1. Fetch real package from proxy
echo "Fetching real package (github.com/pkg/errors) via proxy..."
go get github.com/pkg/errors@v0.9.1

# 2. Fetch dummy package from local via virtual repo
echo "Fetching dummy package via virtual repo..."
go get ${MODULE}@${VERSION}

echo "Cleaning up..."
rm -rf "$WORKDIR"
rm -rf "$PROJECT_ROOT/tests/integration/golang/workspace"

echo "Integration test completed successfully!"
