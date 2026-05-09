#!/usr/bin/env node
import { createHash } from "node:crypto";
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  statSync,
  writeFileSync,
} from "node:fs";
import { basename, join } from "node:path";

const assetDir = "release-assets";
const packageJson = JSON.parse(readFileSync("package.json", "utf8"));
const version = process.env.CIFENCE_RELEASE_SEMVER || packageJson.version;
const tag = process.env.CIFENCE_RELEASE_TAG || `v${version}`;

const binaries = [
  {
    source: "dist/bin/linux-x64/cifence",
    asset: "cifence-linux-amd64",
  },
  {
    source: "dist/bin/linux-arm64/cifence",
    asset: "cifence-linux-arm64",
  },
  {
    source: "dist/bin/darwin-x64/cifence",
    asset: "cifence-darwin-amd64",
  },
  {
    source: "dist/bin/darwin-arm64/cifence",
    asset: "cifence-darwin-arm64",
  },
  {
    source: "dist/bin/win32-x64/cifence.exe",
    asset: "cifence-windows-amd64.exe",
  },
];

mkdirSync(assetDir, { recursive: true });

for (const binary of binaries) {
  assertFile(binary.source);
  copyFileSync(binary.source, join(assetDir, binary.asset));
}

assertFile(join(assetDir, "cifence-sbom.spdx.json"));

const subjects = listAssets()
  .filter((name) => name !== "checksums.txt" && name !== "provenance.json")
  .map((name) => {
    const path = join(assetDir, name);
    return {
      name,
      sha256: sha256(path),
      size: statSync(path).size,
    };
  });

const provenance = {
  predicateType: "https://slsa.dev/provenance/v1",
  builder: "github-actions",
  repository: process.env.GITHUB_REPOSITORY || "oaslananka-lab/cifence",
  ref: process.env.GITHUB_REF || "",
  sha: process.env.GITHUB_SHA || "",
  tag,
  version,
  subjects,
};
writeFileSync(join(assetDir, "provenance.json"), `${JSON.stringify(provenance, null, 2)}\n`, {
  mode: 0o600,
});

const checksumLines = listAssets()
  .filter((name) => name !== "checksums.txt")
  .map((name) => `${sha256(join(assetDir, name))}  ${name}`)
  .sort();
writeFileSync(join(assetDir, "checksums.txt"), `${checksumLines.join("\n")}\n`, { mode: 0o600 });

process.stdout.write("release assets prepared\n");

function listAssets() {
  return binaries
    .map((binary) => binary.asset)
    .concat(["cifence-sbom.spdx.json", "provenance.json"]);
}

function assertFile(path) {
  if (!existsSync(path) || statSync(path).size <= 0) {
    throw new Error(`${path} is missing or empty`);
  }
}

function sha256(path) {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}
