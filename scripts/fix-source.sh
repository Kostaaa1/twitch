#!/usr/bin/env bash
#
# fix-source.sh — fix jittery/awkward concatenated Twitch VOD source segments.
#
# What it does:
#   - Probes the real framerate (no hardcoded 60fps assumption).
#   - If the stream is constant-rate at an integer fps, snaps PTS/DTS to that
#     frame grid to remove scheduler jitter (frames landing at 16/17ms instead
#     of a steady 16.667ms for 60fps). Lossless: -c copy, timestamps only.
#   - If the stream is VFR or a fractional NTSC rate (59.94/29.97), snapping
#     would corrupt timing, so it just remuxes (still lossless, still fixes the
#     container: faststart mp4).
#
# Usage:
#   scripts/fix-source.sh input.ts [output.mp4]
#
set -euo pipefail

in="${1:?usage: fix-source.sh input.ts [output.mp4]}"
out="${2:-${in%.*}.mp4}"

if [[ "$in" == "$out" ]]; then
  echo "input and output are the same path; pick a different output" >&2
  exit 1
fi

# r_frame_rate = base/exact rate, avg_frame_rate = measured average.
# They disagree => variable frame rate.
r_rate=$(ffprobe -v0 -select_streams v:0 -show_entries stream=r_frame_rate  -of csv=p=0 "$in")
a_rate=$(ffprobe -v0 -select_streams v:0 -show_entries stream=avg_frame_rate -of csv=p=0 "$in")

num=${r_rate%/*}
den=${r_rate#*/}

remux_only() {
  echo "[$1] remuxing without restamping"
  ffmpeg -y -i "$in" -c copy -movflags +faststart "$out"
}

# Bail to plain remux when snapping is unsafe:
#   - VFR (measured avg != declared rate)
#   - fractional rate (den != 1, e.g. 60000/1001 NTSC)
#   - unparseable / zero
if [[ "$r_rate" != "$a_rate" ]]; then
  remux_only "VFR: r=$r_rate avg=$a_rate"
  exit 0
fi
if [[ "$den" != "1" || -z "$num" || "$num" == "0" ]]; then
  remux_only "non-integer fps: $r_rate"
  exit 0
fi

# grid = one frame period in the 90kHz MPEG-TS timebase
grid=$(( 90000 / num ))
echo "[snap] ${num}fps -> grid ${grid} ticks (90kHz)"

ffmpeg -y -i "$in" \
  -c copy \
  -bsf:v "setts=pts=round((PTS-STARTPTS)/${grid})*${grid}+STARTPTS:dts=round((DTS-STARTDTS)/${grid})*${grid}+STARTDTS" \
  -movflags +faststart \
  "$out"

echo "wrote $out"
