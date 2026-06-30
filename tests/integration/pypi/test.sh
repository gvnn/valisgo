#!/usr/bin/env bash
set -e

# Change to project root
cd "$(dirname "$0")/../../.."
PROJECT_ROOT=$(pwd)

REGISTRY_URL="http://localhost:8080/registries/mypypi/repositories/myrepo"
PKG_DIR="tests/integration/pypi/dummy_pkg"

echo "Building python package..."
cd "$PKG_DIR"
rm -rf build dist *.egg-info venv .venv

python3 -m venv .venv
source .venv/bin/activate
pip install build twine

python3 -m build

echo "Uploading package using twine..."
twine upload --repository-url "${REGISTRY_URL}/" -u dummy -p dummy dist/*

echo "Installing package using pip..."
python3 -m venv venv
./venv/bin/pip install --index-url "${REGISTRY_URL}/simple/" dummy-pkg-integration

echo "Integration test completed successfully!"
