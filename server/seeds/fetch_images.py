"""Download the curated Forher demo images; leave existing valid files intact."""

import argparse
import json
from pathlib import Path
from urllib.parse import urlsplit
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parent
MAX_BYTES = 20 * 1024 * 1024


def is_webp(data):
    return len(data) > 12 and data[:4] == b"RIFF" and data[8:12] == b"WEBP"


def fetch(manifest_path, assets_dir):
    manifest = json.loads(manifest_path.read_text())
    assets_dir = assets_dir.resolve()
    for product in manifest["products"]:
        for image in product["images"]:
            target = (assets_dir / image["file"]).resolve()
            if not target.is_relative_to(assets_dir):
                raise ValueError("Image path escapes assets directory")
            if target.is_file() and is_webp(target.read_bytes()):
                print(f"Cached {image['file']}")
                continue
            source = urlsplit(image["source_url"])
            if source.scheme != "https" or source.netloc != "forhershop.vn":
                raise ValueError("Only HTTPS images from forhershop.vn are allowed")
            request = Request(image["source_url"], headers={
                "User-Agent": "Aodora-Demo-Seeder/1.0",
                "Referer": product["source_url"],
            })
            with urlopen(request, timeout=45) as response:
                if urlsplit(response.url).netloc != "forhershop.vn":
                    raise ValueError("Unexpected image redirect")
                data = response.read(MAX_BYTES + 1)
            if len(data) > MAX_BYTES or not is_webp(data):
                raise ValueError(f"Invalid or oversized WebP: {image['file']}")
            target.parent.mkdir(parents=True, exist_ok=True)
            temporary = target.with_suffix(".part")
            temporary.write_bytes(data)
            temporary.replace(target)
            print(f"Downloaded {image['file']} ({len(data)} bytes)")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, default=ROOT / "forher.json")
    parser.add_argument("--assets", type=Path, default=ROOT / "assets")
    args = parser.parse_args()
    fetch(args.manifest, args.assets)
