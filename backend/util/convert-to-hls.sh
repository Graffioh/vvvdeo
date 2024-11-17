#!/bin/bash

# Make sure the output folder exists
mkdir -p "../video-hls"

# Generate the HLS streams for 360p, 480p, and 720p
ffmpeg -i ../dio-zawarudo.mp4 \
    -vf "scale=w=640:h=360" -c:v libx264 -b:v 800k -c:a aac -ar 44100 -b:a 96k -hls_time 10 -hls_segment_filename "../video-hls/360p_%03d.ts" -hls_playlist_type vod ../video-hls/360p.m3u8 \
    -vf "scale=w=854:h=480" -c:v libx264 -b:v 1200k -c:a aac -ar 44100 -b:a 128k -hls_time 10 -hls_segment_filename "../video-hls/480p_%03d.ts" -hls_playlist_type vod ../video-hls/480p.m3u8 \
    -vf "scale=w=1280:h=720" -c:v libx264 -b:v 2500k -c:a aac -ar 44100 -b:a 128k -hls_time 10 -hls_segment_filename "../video-hls/720p_%03d.ts" -hls_playlist_type vod ../video-hls/720p.m3u8

# Master file for adaptive res
echo "#EXTM3U
#EXT-X-STREAM-INF:BANDWIDTH=800000,RESOLUTION=640x360
360p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1200000,RESOLUTION=854x480
480p.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2500000,RESOLUTION=1280x720
720p.m3u8" > ../video-hls/master.m3u8
