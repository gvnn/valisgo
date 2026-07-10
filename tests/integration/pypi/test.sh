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

PROXY_URL="http://localhost:8080/registries/mypypi/repositories/pypi-proxy"
VIRTUAL_URL="http://localhost:8080/registries/mypypi/repositories/pypi-virtual"

echo "Installing real package from proxy..."
python3 -m venv proxy-venv
./proxy-venv/bin/pip install --index-url "${PROXY_URL}/simple/" is-odd

echo "Installing real package from virtual..."
python3 -m venv virtual-venv
./virtual-venv/bin/pip install --index-url "${VIRTUAL_URL}/simple/" is-even

echo "Integration test completed successfully!"
