# twitch

A command-line tool for downloading stuff from Twitch (and Kick). VODs, clips, highlights, livestreams. It also has a terminal chat client, still a work in progress.

## Build

Requires Go 1.27+.

```sh
go build -o twitch ./app
```

## Downloading

Pass a URL, a VOD ID, a clip slug or a channel name:

```sh
./twitch download 2926188923 # downloads the vod
./twitch download DeliciousDelightfulPicklesWOOP # downloads the clip
./twitch download xqc                 # records the live stream
```

Full links work too, e.g. `https://www.twitch.tv/videos/2926188923` or `https://www.twitch.tv/xqc`. If a VOD link has a `?t=1h2m3s` timestamp, the download starts from there.

You can pass several at once, and they download in parallel. Use `-t` to limit how many run at the same time.

Kick works the same way. Anything with `kick.com` in it (or a Kick UUID) is sent to the Kick downloader.

### Flags

| Flag | What it does |
|------|--------------|
| `-q, --quality` | `best` (default), `worst`, `1080p`, `720p`, `480p`, `360p`, `160p`, `audio_only` |
| `-o, --output` | A directory, or a full file path. Defaults to the output dir in your config. One of the two has to be set |
| `-s, --start` / `-e, --end` | Download only part of a VOD, e.g. `-s 1h5m -e 2h30m` |
| `-t, --threads` | Max number of downloads running at once |
| `-w, --watch` | Wait for channels to go live and start recording automatically |
| `-v` | More logging (`-vv`, `-vvv` for even more) |

### Whole channels

Grab the latest VODs or top clips from a channel:

```sh
./twitch download --channels=xqc,forsen --limit-videos=5
./twitch download --channels=xqc --limit-clips=20 --filter-clips-by=last_week
```

- `--filter-clips-by`: `last_day`, `last_week`, `last_month`, `all_time`
- `--filter-videos-by`: `archive`, `highlight`, `upload`
- `--sort-videos-by`: `time`, `views`

### Batch file

If you download the same things often, put them in a JSON file and pass the path instead. Each entry can have its own quality, output and timestamps:

```json
[
  { "input": "xqc", "quality": "720p", "output": "~/streams" },
  { "input": "2926188923", "quality": "best", "start": "1h5m20s", "end": "2h5m30s" }
]
```

```sh
./twitch download runner.json
```

See `runner.example.json` for more examples.

### Watching channels

`--watch` uses Twitch EventSub to start recording as soon as a channel goes live, so you can leave it running in the background. It needs a Twitch login, see [Authentication](#authentication).

## Other commands

```sh
./twitch info xqc       # channel info with recent videos and clips
./twitch config         # print the current config
./twitch config path    # where the config lives
./twitch config init    # reset to defaults
./twitch config rm      # delete the config dir
```

The config lives at `twitch/twitch_config.json` inside your OS config directory (`~/.config` on Linux, `~/Library/Application Support` on macOS). It holds the default output dir, spinner settings, your OAuth tokens and the chat colors.

## Authentication

Downloading and `info` work without an account. Chat and `--watch` need you to log in, because Twitch's IRC chat, EventSub and Helix API all require a user token.

For now you have to register your own Twitch app:

1. Create an app at [dev.twitch.tv/console](https://dev.twitch.tv/console/apps). Set the OAuth redirect URL to something local, like `http://localhost:3000/callback`.
2. Run `./twitch config init` if you don't have a config yet, then fill in `oauth_credentials` in the config file:

   ```json
   "oauth_credentials": {
     "client_id": "...",
     "client_secret": "...",
     "redirect_url": "http://localhost:3000/callback"
   }
   ```

3. Run `./twitch chat` once to log in. It prints a link; open it in your browser and approve. The tool listens on the redirect URL's port, receives the redirect and saves the tokens to the config. After that, it refreshes them on its own.

## Chat

`./twitch chat` opens a terminal chat client. It needs a login (see above) and remembers which chats you had open.

| Key | |
|-----|--|
| `enter` | Send message |
| `tab` / `ctrl+→` | Next chat |
| `shift+tab` / `ctrl+←` | Previous chat |
| `ctrl+shift+→` / `ctrl+shift+←` | Move the current tab |
| `ctrl+w` | Close the current chat |
| `ctrl+o` | Open the stream in VLC (macOS only for now) |
| `ctrl+g` | Toggle the help panel |
| `esc` / `ctrl+c` | Quit |

Type `/add <channel>` to open another chat.

## Notes

- Twitch mutes some VOD segments because of DMCA. The downloader tries to fetch the unmuted versions first, which often works if the VOD is recent.
- Downloading audio-only clips needs `ffmpeg` on your PATH.

## Planned

- Resuming interrupted downloads
- Burning chat into the video
- Finishing the chat client (include moderator)
- File naming templates (see below)

### File naming templates

Right now files are named after the title. The plan is to let you choose the filename format with a template per media type, set in the config:

```json
"naming_templates": {
  "video": "{channel}/{date}_{name}_{quality}",
  "clip": "{channel}/clips/{name}",
  "stream": "{channel}/live_{date}"
}
```

Placeholders:

| Placeholder | Becomes |
|-------------|---------|
| `{channel}` | Channel name |
| `{name}` | Title, cut to 60 characters |
| `{date}` | When the VOD, clip or stream was created (`2026-11-23`) |
| `{quality}` | Quality, e.g. `1080p60` |
| `{id}` | VOD ID or clip slug |

How it'll work:

- The template is filled in after the metadata is fetched, just before the file is created. The extension (`.ts`, `.mp4`) is added automatically.
- The result is placed inside the output dir. A `/` in the template creates subfolders, so you can sort downloads by channel.
- Characters that aren't allowed in filenames (`/ \ : * ? " < > |`) are stripped from titles and channel names, so a title can't create extra folders by accident.
- If `-o` is a full file path, that wins and the template is ignored. With no template set, the current title-based naming is kept.
- A per-run `--template` flag can override the config.