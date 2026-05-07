## Features
- downloading twitch clips, VODs, highlights
- record livestreams
- partial downloading VODs by timestamps
- download by specific qualities (default is highest/best)
- download chat
- audio only (ffmpeg used for clips?)
- avoid ffmpeg (used only for converting clip video to audio?)
- twitch auth
- cli chatroom (no emotes, only text displayed)
- kick support

## TODOS:
- daemon for auto-recording: usage of event-sub (websockets/webhooks) or polling to discover when specified channels go live, and record them to configured path
- publishing of the clips (maybe to multiple platforms, a lot of people are reposting them to TikTok, youtube shorts etc.)
- learn fyne: chatterino clone - display 7TV emotes?
- chat overlay over vod (realtime rendering for ivestreams??) - do this without ffmpeg?
- auto clipper: automatic clipping based on chat interaction
	* hard to do (how to detect that something funnt happened)
	* listening to extreme change in rate of the messages that are being sent
	* analyze messages, if emotes, high possibility of something funny happened
	* problem: channel specific emote

## IDEAS

Stream tooling
1. M3U8 inspector CLI — like dig but for HLS playlists. Shows segment durations, bitrates, AES keys, discontinuities, gaps, drift between variants. Surprisingly useful and not many good ones exist.
2. HLS restreamer — pull from one source, push to N destinations. Mini self-hosted Restream.io. Goes well with Go's concurrency.
3. Local DVR server — buffers any live HLS feed on disk so you can pause/rewind. Serves a modified playlist back to your player.
4. LL-HLS converter — take a normal HLS stream and repackage it as Low-Latency HLS with partial segments.

Twitch-flavored extensions of what you built
1. Chat-synced VOD archiver — download VOD + chat, render chat as overlay or burn it into the video at correct timestamps.
2. Auto-clipper — watch a live stream, detect chat spikes or "+2"/"LUL" bursts, save the preceding 30s as a clip.
3. Highlight detector — audio loudness + Whisper transcription to auto-cut highlights from a long VOD.

More ambitious
1. Searchable VOD archive — download streams, transcribe with Whisper, full-text index. "Find the moment X said Y."
2. HLS load tester — simulate N concurrent viewers, measure stalls, bitrate switches, segment fetch times. Good for testing your own streaming setup.
3. Multi-stream viewer — watch 4 streams in one window, synced or independent, with audio routing.