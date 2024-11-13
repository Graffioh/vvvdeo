import { VideoPlayer } from "./videoPlayer.js";

const backendUrl = "http://localhost:8080";

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

  const videoElement = document.getElementById("shaka-player-video");
  const coordinates = [];
  const labels = [];

  videoElement.addEventListener("contextmenu", (event) => {
    const rect = videoElement.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;
    console.log("X: " + x);
    console.log("Y: " + y);

    coordinates.push({ x, y });
    labels.push(1);
  });

  videoElement.addEventListener("auxclick", (event) => {
    const rect = videoElement.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;
    console.log("X label: " + x);
    console.log("Y label: " + y);

    coordinates.push({ x, y });
    labels.push(0);
  });

  const inferenceButtonElement = document.getElementById("inference-btn");

  inferenceButtonElement.addEventListener("click", () => {
    fetch(backendUrl + "/inference", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ coordinates: coordinates, labels: labels }),
    })
      .then((response) => response.json())
      .then((data) => {
        console.log("result:", data);

        const segmentedFrame = document.getElementById("segmented-frame");
        segmentedFrame.src = "./sam2-try/" + data.segmented_image_path;
        segmentedFrame.style.display = "block";

        const imageContainer = document.getElementById("image-container");
        segmentedImages = data.segmented_image_paths;
        segmentedImages.forEach((imgName) => {
          const img = new Image();
          img.src = "./sam2-try/" + imgName;
          img.alt = imgName;
          imageContainer.appendChild(img);
        });
      })
      .catch((error) => console.error("Error:", error));
  });

  const inferenceFramesButtonElement = document.getElementById(
    "inference-frames-btn",
  );

  inferenceFramesButtonElement.addEventListener("click", () => {
    fetch(backendUrl + "/inference-frames", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ coordinates: coordinates, labels: labels }),
    })
      .then((response) => response.json())
      .then((data) => {
        console.log("result:", data);

        const imageContainer = document.getElementById("image-container");
        segmentedImages = data.segmented_image_paths;
        segmentedImages.forEach((imgName) => {
          const img = new Image();
          img.src = "./sam2-try/" + imgName;
          img.alt = imgName;
          imageContainer.appendChild(img);
        });
      })
      .catch((error) => console.error("Error:", error));
  });
});
