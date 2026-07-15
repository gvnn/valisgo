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

PROXY_URL="${PROXY_URL:-http://localhost:8080/registries/mynpm/repositories/npm-proxy/}"
LOCAL_URL="${LOCAL_URL:-http://localhost:8080/registries/mynpm/repositories/npm-local/}"
VIRTUAL_URL="${VIRTUAL_URL:-http://localhost:8080/registries/mynpm/repositories/npm-virtual/}"

echo "Publishing dummy package..."
cd tests/integration/npm/dummy-pkg
npm version patch
npm publish --registry="$LOCAL_URL"

echo "Installing real package from proxy..."
mkdir -p ../workspace && cd ../workspace
rm -rf node_modules package.json package-lock.json
npm init -y

# We will test npm install using the proxy
npm install is-odd --registry="$PROXY_URL"

echo "Installing dummy package from local..."
npm install my-dummy-pkg --registry="$LOCAL_URL"

echo "Installing packages from virtual..."
mkdir -p ../virtual-workspace && cd ../virtual-workspace
rm -rf node_modules package.json package-lock.json
npm init -y
npm install is-number my-dummy-pkg --registry="$VIRTUAL_URL"

echo "Integration test completed successfully!"
