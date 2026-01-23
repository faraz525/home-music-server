#!/usr/bin/env python3
"""
SoundCloud Likes Downloader

Downloads a user's liked tracks from SoundCloud as MP3 files.
Outputs a manifest.json with metadata for each downloaded track.

Usage: python3 downloader.py <oauth_token> <output_dir>

Requires: yt-dlp, ffmpeg
"""

import sys
import json
import os
from pathlib import Path

def download_soundcloud_likes(oauth_token: str, output_dir: str) -> int:
    """Download SoundCloud likes as MP3 using yt-dlp"""
    try:
        import yt_dlp
    except ImportError:
        print("Error: yt-dlp not installed. Run: pip install yt-dlp", file=sys.stderr)
        return 1

    Path(output_dir).mkdir(parents=True, exist_ok=True)

    ydl_opts = {
        'format': 'bestaudio/best',
        'postprocessors': [{
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'mp3',
            'preferredquality': '320',
        }],
        'outtmpl': os.path.join(output_dir, '%(id)s.%(ext)s'),
        'quiet': False,
        'no_warnings': False,
        'extract_flat': False,
        'ignoreerrors': True,
        'http_headers': {
            'Authorization': f'OAuth {oauth_token}'
        },
        'cookiesfrombrowser': None,
    }

    manifest = []

    try:
        with yt_dlp.YoutubeDL(ydl_opts) as ydl:
            print(f"Fetching SoundCloud likes...")

            result = ydl.extract_info(
                'https://soundcloud.com/you/likes',
                download=True
            )

            if result is None:
                print("No results returned from SoundCloud", file=sys.stderr)
                write_manifest(output_dir, manifest)
                return 0

            entries = result.get('entries', [result]) if 'entries' in result else [result]

            for entry in entries:
                if entry is None:
                    continue

                track_id = entry.get('id', '')
                title = entry.get('title', 'Unknown')
                uploader = entry.get('uploader', entry.get('artist', 'Unknown'))
                duration = entry.get('duration', 0)

                expected_file = os.path.join(output_dir, f'{track_id}.mp3')

                if os.path.exists(expected_file):
                    manifest.append({
                        'file_path': expected_file,
                        'title': title,
                        'artist': uploader,
                        'duration': duration or 0,
                        'soundcloud_id': str(track_id)
                    })
                    print(f"  ✓ {title} by {uploader}")
                else:
                    for f in os.listdir(output_dir):
                        if f.startswith(str(track_id)) and f.endswith('.mp3'):
                            manifest.append({
                                'file_path': os.path.join(output_dir, f),
                                'title': title,
                                'artist': uploader,
                                'duration': duration or 0,
                                'soundcloud_id': str(track_id)
                            })
                            print(f"  ✓ {title} by {uploader}")
                            break

        write_manifest(output_dir, manifest)
        print(f"\nDownloaded {len(manifest)} tracks")
        return 0

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        write_manifest(output_dir, manifest)
        return 1


def write_manifest(output_dir: str, manifest: list):
    """Write manifest.json file"""
    manifest_path = os.path.join(output_dir, 'manifest.json')
    with open(manifest_path, 'w') as f:
        json.dump(manifest, f, indent=2)


def main():
    if len(sys.argv) != 3:
        print("Usage: downloader.py <oauth_token> <output_dir>")
        print("\nTo get your OAuth token:")
        print("1. Open SoundCloud in your browser")
        print("2. Open Developer Tools (F12)")
        print("3. Go to Application/Storage > Cookies > soundcloud.com")
        print("4. Find the 'oauth_token' cookie and copy its value")
        sys.exit(1)

    oauth_token = sys.argv[1]
    output_dir = sys.argv[2]

    sys.exit(download_soundcloud_likes(oauth_token, output_dir))


if __name__ == '__main__':
    main()
