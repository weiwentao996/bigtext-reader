#!/usr/bin/env bash
set -euo pipefail

TAG="${1:-$(git describe --tags --abbrev=0)}"
VERSION="${TAG#v}"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid release tag '$TAG'. Expected format like v1.2.3." >&2
  exit 1
fi

VERSION="$VERSION" node <<'NODE'
const fs = require("fs");

const path = "wails.json";
const config = JSON.parse(fs.readFileSync(path, "utf8"));

config.info = {
  ...(config.info || {}),
  companyName: config.info?.companyName || config.author?.name || config.name,
  productName: config.info?.productName || config.name,
  productVersion: process.env.VERSION,
};

fs.writeFileSync(path, JSON.stringify(config, null, 2) + "\n");
NODE

export VITE_APP_VERSION="$VERSION"
wails build -clean
