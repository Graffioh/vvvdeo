import { VideoPlayer } from "./videoPlayer.js";

// used for conversion to HLS or MPEG-DASH
function waitForVideoAvailability(manifestUri, videoPlayer) {
  const loadingMessage = document.getElementById("loading-msg");
  const checkInterval = 2000;

  const checkVideo = async () => {
    try {
      const response = await fetch(manifestUri, { method: "HEAD" });
      if (response.ok) {
        console.log("Video manifest found. Initializing player...");
        await videoPlayer.loadManifest(manifestUri);
        videoPlayer.setVideoPlayerVisible();
        loadingMessage.hidden = true;
      } else {
        console.log("Waiting for video conversion...");
        setTimeout(checkVideo, checkInterval);
      }
    } catch (error) {
      console.log("Error checking video availability:", error);
      setTimeout(checkVideo, checkInterval);
    }
  };

  loadingMessage.hidden = false;
  checkVideo();
}

document.addEventListener("DOMContentLoaded", () => {
  // VIDEO CONVERSION
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
  const manifestUris = {
    mp4: "http://localhost:8080/zawarudo.mp4",
    hls: "http://localhost:8080/zawarudo/master.m3u8",
    dash: "http://localhost:8080/zawarudo/my_video_manifest.mpd",
  };

  const videoPlayer = new VideoPlayer();
  const mp4Btn = document.getElementById("mp4-btn");
  const hlsBtn = document.getElementById("hls-btn");
  const dashBtn = document.getElementById("dash-btn");

  mp4Btn.addEventListener("click", async () => {
    await videoPlayer.loadManifest(manifestUris.mp4);
    videoPlayer.setVideoPlayerVisible();
  });

  hlsBtn.addEventListener("click", async () => {
    waitForVideoAvailability(manifestUris.hls, videoPlayer);
  });

  dashBtn.addEventListener("click", async () => {
    waitForVideoAvailability(manifestUris.dash, videoPlayer);
  });
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
});
