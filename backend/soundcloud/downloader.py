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
import ssl
import urllib.request
import urllib.error
from pathlib import Path

# Create SSL context that works on macOS
ssl_context = ssl.create_default_context()
try:
    import certifi
    ssl_context.load_verify_locations(certifi.where())
except ImportError:
    ssl_context.check_hostname = False
    ssl_context.verify_mode = ssl.CERT_NONE


def get_soundcloud_likes(oauth_token: str, limit: int = 200) -> list:
    """Fetch liked tracks using SoundCloud API"""

    # First get user info to get user_id
    me_url = f"https://api-v2.soundcloud.com/me?oauth_token={oauth_token}"

    try:
        req = urllib.request.Request(me_url)
        req.add_header('User-Agent', 'Mozilla/5.0')
        with urllib.request.urlopen(req, timeout=30, context=ssl_context) as response:
            user_data = json.loads(response.read().decode())
            user_id = user_data.get('id')
            username = user_data.get('username', 'Unknown')
            print(f"Authenticated as: {username} (ID: {user_id})")
    except urllib.error.HTTPError as e:
        print(f"Failed to authenticate: HTTP {e.code}", file=sys.stderr)
        if e.code == 401:
            print("OAuth token may be expired or invalid", file=sys.stderr)
        return []
    except Exception as e:
        print(f"Failed to get user info: {e}", file=sys.stderr)
        return []

    # Fetch likes
    likes_url = f"https://api-v2.soundcloud.com/users/{user_id}/track_likes?limit={limit}&oauth_token={oauth_token}"

    try:
        req = urllib.request.Request(likes_url)
        req.add_header('User-Agent', 'Mozilla/5.0')
        with urllib.request.urlopen(req, timeout=30, context=ssl_context) as response:
            likes_data = json.loads(response.read().decode())

        tracks = []
        for item in likes_data.get('collection', []):
            track = item.get('track')
            if track:
                tracks.append({
                    'id': track.get('id'),
                    'title': track.get('title', 'Unknown'),
                    'artist': track.get('user', {}).get('username', 'Unknown'),
                    'duration': track.get('duration', 0) // 1000,  # Convert ms to seconds
                    'permalink_url': track.get('permalink_url'),
                })

        print(f"Found {len(tracks)} liked tracks")
        return tracks

    except urllib.error.HTTPError as e:
        print(f"Failed to fetch likes: HTTP {e.code}", file=sys.stderr)
        return []
    except Exception as e:
        print(f"Failed to fetch likes: {e}", file=sys.stderr)
        return []


def download_tracks(tracks: list, output_dir: str, max_tracks: int = 50) -> list:
    """Download tracks as MP3 using yt-dlp"""
    try:
        import yt_dlp
    except ImportError:
        print("Error: yt-dlp not installed. Run: pip install yt-dlp", file=sys.stderr)
        return []

    Path(output_dir).mkdir(parents=True, exist_ok=True)
    manifest = []

    # Limit number of tracks per sync to avoid overwhelming
    tracks_to_download = tracks[:max_tracks]

    for i, track in enumerate(tracks_to_download):
        url = track.get('permalink_url')
        if not url:
            continue

        track_id = track.get('id')
        title = track.get('title', 'Unknown')
        artist = track.get('artist', 'Unknown')

        print(f"[{i+1}/{len(tracks_to_download)}] Downloading: {title} by {artist}")

        ydl_opts = {
            'format': 'bestaudio/best',
            'postprocessors': [{
                'key': 'FFmpegExtractAudio',
                'preferredcodec': 'mp3',
                'preferredquality': '320',
            }],
            'outtmpl': os.path.join(output_dir, f'{track_id}.%(ext)s'),
            'quiet': True,
            'no_warnings': True,
            'ignoreerrors': True,
        }

        try:
            with yt_dlp.YoutubeDL(ydl_opts) as ydl:
                ydl.download([url])

            # Check if file was downloaded
            expected_file = os.path.join(output_dir, f'{track_id}.mp3')
            if os.path.exists(expected_file):
                manifest.append({
                    'file_path': expected_file,
                    'title': title,
                    'artist': artist,
                    'duration': track.get('duration', 0),
                    'soundcloud_id': str(track_id)
                })
                print(f"  ✓ Downloaded successfully")
            else:
                print(f"  ✗ File not found after download")

        except Exception as e:
            print(f"  ✗ Failed: {e}")
            continue

    return manifest


def write_manifest(output_dir: str, manifest: list):
    """Write manifest.json file"""
    Path(output_dir).mkdir(parents=True, exist_ok=True)
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

    # Get liked tracks from API
    tracks = get_soundcloud_likes(oauth_token)

    if not tracks:
        print("No tracks to download")
        write_manifest(output_dir, [])
        sys.exit(0)

    # Download tracks
    manifest = download_tracks(tracks, output_dir)

    # Write manifest
    write_manifest(output_dir, manifest)

    print(f"\nCompleted: {len(manifest)} tracks downloaded")
    sys.exit(0)


if __name__ == '__main__':
    main()
