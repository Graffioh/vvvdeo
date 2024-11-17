#!/bin/bash

mkdir -p "../video-dash"

if [ -z "$1" ]; then
  echo "Usage: $0 {audio|video-160|video-320|video-640|video-1280|manifest|all}"
  exit 1
fi

case "$1" in
  audio)
    echo "Generating audio file..."
    ffmpeg -i ../dio-zawarudo.mp4 -vn -acodec libvorbis -ab 128k -dash 1 ../video-dash/my_audio.webm
    ;;
  
  video-160)
    echo "Generating 160x90 video file..."
    ffmpeg -i ../dio-zawarudo.mp4 -preset ultrafast -c:v libvpx-vp9 -keyint_min 150 -g 150 -tile-columns 4 -frame-parallel 1 \
      -an -vf scale=160:90 -b:v 250k -f webm -dash 1 ../video-dash/video_160x90_250k.webm
    ;;
  
  video-320)
    echo "Generating 320x180 video file..."
    ffmpeg -i ../dio-zawarudo.mp4 -preset ultrafast -c:v libvpx-vp9 -keyint_min 150 -g 150 -tile-columns 4 -frame-parallel 1 \
      -an -vf scale=320:180 -b:v 500k -f webm -dash 1 ../video-dash/video_320x180_500k.webm
    ;;
  
  video-640)
    echo "Generating 640x360 video files..."
    ffmpeg -i ../dio-zawarudo.mp4 -preset ultrafast -c:v libvpx-vp9 -keyint_min 150 -g 150 -tile-columns 4 -frame-parallel 1 \
      -an -vf scale=640:360 -b:v 750k -f webm -dash 1 ../video-dash/video_640x360_750k.webm
    ;;
  
  video-1280)
    echo "Generating 1280x720 video file..."
    ffmpeg -i ../dio-zawarudo.mp4 -preset ultrafast -c:v libvpx-vp9 -keyint_min 150 -g 150 -tile-columns 4 -frame-parallel 1 \
      -an -vf scale=1280:720 -b:v 1500k -f webm -dash 1 ../video-dash/video_1280x720_1500k.webm
    ;;
  
  manifest)
    echo "Generating DASH manifest..."
    ffmpeg \
      -f webm_dash_manifest -i ../video-dash/video_160x90_250k.webm \
      -f webm_dash_manifest -i ../video-dash/video_320x180_500k.webm \
      -f webm_dash_manifest -i ../video-dash/video_640x360_750k.webm \
      -f webm_dash_manifest -i ../video-dash/video_1280x720_1500k.webm \
      -f webm_dash_manifest -i ../video-dash/my_audio.webm \
      -c copy \
      -map 0 -map 1 -map 2 -map 3 -map 4 \
      -f webm_dash_manifest \
      -adaptation_sets "id=0,streams=0,1,2,3 id=1,streams=4" \
      ../video-dash/my_video_manifest.mpd
    ;;

  all)
    echo "Generating all files..."
    $0 audio
    $0 video-160
    $0 video-320
    $0 video-640
    $0 video-1280
    $0 manifest
    ;;

  *)
    echo "Invalid option. Use {audio|video-160|video-320|video-640|video-1280|manifest|all}"
    exit 1
    ;;
esac

