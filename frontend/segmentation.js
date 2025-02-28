import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { FFmpeg } from "@ffmpeg/ffmpeg";

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL;
//const BACKEND_WS_URL = import.meta.env.VITE_BACKEND_WS_URL;

const videoPlayer = document.getElementById("video-player");
const videoInputUpload = document.getElementById("input-video");
const videoPlayerContainer = document.getElementById("video-player-container");
const loadingSpinnerContainer = document.getElementById(
  "loading-spinner-container",
);
const ffmpegMessage = document.getElementById("ffmpeg-message");

// random filename generation
const adjectives = [
  "funky",
  "silly",
  "wacky",
  "quirky",
  "zany",
  "goofy",
  "bouncy",
  "fluffy",
  "giggly",
  "sparkly",
];
const nouns = [
  "unicorn",
  "banana",
  "penguin",
  "ninja",
  "robot",
  "panda",
  "rocket",
  "jellybean",
  "marshmallow",
  "cupcake",
];

function generateRandomFilename() {
  const randomAdjective =
    adjectives[Math.floor(Math.random() * adjectives.length)];
  const randomNoun = nouns[Math.floor(Math.random() * nouns.length)];
  return `${randomAdjective}_${randomNoun}.mp4`;
}

// ffmpeg wasm trimming
const BASE_FFMPEG_WASM_URL = "https://unpkg.com/@ffmpeg/core@0.12.6/dist/esm";

let trimmedVideoFile = null;
let ffmpeg = null;
const trim = async (file, startTrim, endTrim) => {
  loadingSpinnerContainer.style.display = "flex";
  if (ffmpeg === null) {
    ffmpeg = new FFmpeg();
    ffmpeg.on("log", ({ message }) => {
      console.log(message);
    });
    ffmpeg.on("progress", ({ progress, time }) => {
      ffmpegMessage.innerHTML = `${progress * 100} %`;
    });
    await ffmpeg.load({
      coreURL: await toBlobURL(
        `${BASE_FFMPEG_WASM_URL}/ffmpeg-core.js`,
        "text/javascript",
      ),
      wasmURL: await toBlobURL(
        `${BASE_FFMPEG_WASM_URL}/ffmpeg-core.wasm`,
        "application/wasm",
      ),
    });
  }

  const { name } = file;
  await ffmpeg.writeFile(name, await fetchFile(file));
  ffmpegMessage.innerHTML = "Start trimming...";
  await ffmpeg.exec([
    "-fflags",
    "+genpts",
    "-ss",
    startTrim,
    "-to",
    endTrim,
    "-i",
    name,
    "-c:v",
    "copy",
    "output.mp4",
  ]);

  const data = await ffmpeg.readFile("output.mp4");
  trimmedVideoFile = new Blob([data.buffer], { type: "video/mp4" });

  videoPlayer.style.display = "block";
  videoPlayer.src = URL.createObjectURL(trimmedVideoFile);
  loadingSpinnerContainer.style.display = "none";
};

const toggleIframeBtn = document.getElementById("toggle-iframe-btn");
const iframeContainer = document.getElementById("iframe-container");
let iframeLoaded = false;

toggleIframeBtn.addEventListener("click", () => {
  if (!iframeLoaded) {
    const iframe = document.createElement("iframe");
    iframe.src = "https://cobalt.tools/";
    iframe.style.position = "absolute";
    iframe.style.top = "0";
    iframe.style.left = "0";
    iframe.style.width = "100%";
    iframe.style.height = "100%";
    iframeContainer.appendChild(iframe);

    iframeContainer.style.display = "block";
    iframeLoaded = true;
    toggleIframeBtn.classList.add("active");
  } else {
    if (iframeContainer.style.display === "block") {
      iframeContainer.style.display = "none";
      toggleIframeBtn.classList.remove("active");
    } else {
      iframeContainer.style.display = "block";
      toggleIframeBtn.classList.add("active");
    }
  }
});

const downloadButton = document.getElementById("download-btn");

// upload video
videoInputUpload.addEventListener("change", () => {
  const videoFile = videoInputUpload.files[0];

  if (videoFile.type !== "video/mp4") {
    alert("Please upload an MP4 video file only.");
    videoInputUpload.value = "";
    return;
  }

  const videoURL = URL.createObjectURL(videoFile);
  videoPlayer.src = videoURL;
  videoPlayerContainer.style.display = "block";
  downloadButton.style.display = "flex";
});

downloadButton.addEventListener("click", () => {
  const a = document.createElement("a");
  a.href = videoPlayer.src;
  a.download = "crafted_vvvdeo.mp4";
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(videoPlayer.src);
  a.remove();
});

// const convertStreamToFile = async () => {
// try {
// const videoUrl = videoPlayer.src;
// const response = await fetch(videoUrl);
// if (!response.ok) {
// throw new Error(`Failed to fetch video: ${response.statusText}`);
// }
// const videoBlob = await response.blob();
// const videoFile = new File([videoBlob], "streamedVideo.mp4", {
// type: videoBlob.type,
// });
// return videoFile;
// } catch (error) {
// console.error("Error converting video stream to file:", error);
// alert("Failed to convert video stream to file.");
// return null;
// }
// };

// ffmpeg functionalities
const ffmpegInputsContainer = document.getElementById("ffmpeg-container");
const showTrimButton = document.getElementById("show-trim-btn");
const trimButtonFast = document.getElementById("trim-btn");
const showSpeedupButton = document.getElementById("show-speedup-btn");
const speedupButton = document.getElementById("speedup-btn");
const speedupFactorContainer = document.getElementById(
  "speedup-inputs-container",
);
const speedupFactorInput = document.getElementById("speedup-factor-input");
const showSegmentButton = document.getElementById("show-segment-btn");

// Easter egg to enable the Segment button after 3 vvvdeo header clicks
const vvvdeoHeader = document.getElementById("vvvdeo-header");
let vvvdeoHeaderEasterEggClickCount = 0;
let vvvdeoHeaderEasterEggClicTimeout = null;

vvvdeoHeader.addEventListener("click", () => {
  vvvdeoHeaderEasterEggClickCount++;

  if (vvvdeoHeaderEasterEggClickCount === 1) {
    vvvdeoHeaderEasterEggClicTimeout = setTimeout(() => {
      vvvdeoHeaderEasterEggClickCount = 0;
    }, 1000);
  }

  if (vvvdeoHeaderEasterEggClickCount === 3) {
    clearTimeout(vvvdeoHeaderEasterEggClicTimeout);
    vvvdeoHeaderEasterEggClickCount = 0;

    showSegmentButton.disabled = false;
    showSegmentButton.style.opacity = "1";
    showSegmentButton.style.cursor = "pointer";

    vvvdeoHeader.style.color = "#ff69b4";
    vvvdeoHeader.textContent = "vvvdeo unlocked!";
    setTimeout(() => {
      vvvdeoHeader.style.color = "";
      vvvdeoHeader.textContent = "vvvdeo";
    }, 1500);
  }
});

showTrimButton.addEventListener("click", () => {
  ffmpegInputsContainer.style.display = "flex";

  speedupButton.style.display = "none";
  trimButtonFast.style.display = "block";
  speedupFactorContainer.style.display = "none";

  inferenceContainer.style.display = "none";

  showTrimButton.style.color = "grey";
  showSpeedupButton.style.color = "white";
  showSegmentButton.style.color = "white";
});

showSpeedupButton.addEventListener("click", () => {
  const ffmpegEventSource = new EventSource(BACKEND_URL + "/ffmpeg-events");

  ffmpegEventSource.onmessage = function (event) {
    ffmpegMessage.innerHTML = event.data;
  };
  ffmpegEventSource.onerror = function () {
    setTimeout(() => (ffmpegMessage.innerHTML = ""), 1000);
  };

  ffmpegInputsContainer.style.display = "flex";
  speedupButton.style.display = "block";
  trimButtonFast.style.display = "none";
  speedupFactorContainer.style.display = "block";

  inferenceContainer.style.display = "none";

  showTrimButton.style.color = "white";
  showSpeedupButton.style.color = "grey";
  showSegmentButton.style.color = "white";
});

const inferenceContainer = document.getElementById("inference-container");
showSegmentButton.addEventListener("click", () => {
  ffmpegInputsContainer.style.display = "none";
  speedupButton.style.display = "none";
  trimButtonFast.style.display = "none";
  speedupFactorContainer.style.display = "none";
  inferenceContainer.style.display = "flex";
  showTrimButton.style.color = "white";
  showSpeedupButton.style.color = "white";
  showSegmentButton.style.color = "grey";
});

function timeToSeconds(time) {
  const [hours, minutes, seconds] = time.split(":").map(Number);
  return hours * 3600 + minutes * 60 + seconds;
}

function secondsToTime(seconds) {
  const hrs = Math.floor(seconds / 3600)
    .toString()
    .padStart(2, "0");
  const mins = Math.floor((seconds % 3600) / 60)
    .toString()
    .padStart(2, "0");
  const secs = Math.floor(seconds % 60)
    .toString()
    .padStart(2, "0");
  const millis = Math.floor((seconds % 1) * 1000)
    .toString()
    .padStart(3, "0");
  return `${hrs}:${mins}:${secs}.${millis}`;
}

// timestamp management
const startTimestampInput = document.getElementById("start-timestamp-input");
const endTimestampInput = document.getElementById("end-timestamp-input");
const startTimestampBtn = document.getElementById("start-timestamp-btn");
const endTimestampBtn = document.getElementById("end-timestamp-btn");

startTimestampBtn.addEventListener("click", () => {
  startTimestampInput.value = secondsToTime(videoPlayer.currentTime);
});

endTimestampBtn.addEventListener("click", () => {
  if (!startTimestampInput.value) {
    alert("Please select first the starting time.");
    return;
  }

  // what a ugly code lmao
  if (videoPlayer.currentTime < timeToSeconds(startTimestampInput.value)) {
    alert("Please select a valid starting and ending time");
    return;
  }

  endTimestampInput.value = secondsToTime(videoPlayer.currentTime);
});

async function getVideoFile() {
  let videoInputFile;
  const videoSrc = videoPlayer.src || videoPlayer.querySelector("source")?.src;

  if (videoSrc) {
    try {
      const response = await fetch(videoSrc);
      const videoBlob = await response.blob();
      videoInputFile = new File([videoBlob], "video.mp4", {
        type: videoBlob.type,
      });
    } catch (error) {
      console.error("Failed to fetch the video from <video> element:", error);
      return null;
    }
  } else {
    videoInputFile = videoInputUpload.files[0];

    if (!videoInputFile) {
      console.error("No video source or uploaded file found!");
      return null;
    }
  }

  return videoInputFile;
}

trimButtonFast.addEventListener("click", async () => {
  const startTrimValue = startTimestampInput.value;
  const endTrimValue = endTimestampInput.value;

  const videoInputFile = await getVideoFile();

  if (videoInputFile) {
    await trim(videoInputFile, startTrimValue, endTrimValue);
  }

  console.log("GIVE ME CREDITS FOR INFERENCE");

  startTimestampInput.value = null;
  endTimestampInput.value = null;
});

speedupButton.addEventListener("click", async () => {
  const startTrimValue =
    startTimestampInput.value === "00:00:00.000"
      ? "00:00:00.100"
      : startTimestampInput.value;

  const endTrimValue = endTimestampInput.value;

  const videoInputFile = await getVideoFile();

  if (videoInputFile) {
    try {
      const formData = new FormData();
      formData.append("videoFile", videoInputFile);
      formData.append("startTime", startTrimValue);
      formData.append("endTime", endTrimValue);
      formData.append("speedupFactor", speedupFactorInput.value);

      ffmpegInputsContainer.style.display = "none";
      loadingSpinnerContainer.style.display = "flex";

      const speedupVideoResponse = await fetch(BACKEND_URL + "/video/speedup", {
        method: "POST",
        body: formData,
      });

      if (speedupVideoResponse.ok) {
        const speedupVideoblob = await speedupVideoResponse.blob();
        videoPlayer.src = URL.createObjectURL(speedupVideoblob);
        videoPlayer.load();

        ffmpegInputsContainer.style.display = "flex";
        loadingSpinnerContainer.style.display = "none";
      } else {
        console.error(
          "Error fetching speedup video. Status:",
          speedupVideoResponse.status,
        );
        const errorText = await speedupVideoResponse.text();
        console.error("Error details:", errorText);
      }
    } catch (error) {
      console.error("Error fetching speedup video.", error);
      return;
    }
  }

  console.log("GIVE ME CREDITS FOR INFERENCE");

  startTimestampInput.value = null;
  endTimestampInput.value = null;
});

// VIDEO SEGMENTATION
// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
const inferenceVideoButtonElement = document.getElementById(
  "inference-video-btn",
);
let coordinates = [];
let labels = [];
const labelsContainer = document.getElementById("labels-container");

const addPositiveLabelButton = document.getElementById("seg-add");
const addNegativeLabelButton = document.getElementById("seg-excl");
let isVideoPlayable = true;

// add negative or positive points for segmentation
function addPoint(event, shapeType, label) {
  const videoRect = videoPlayer.getBoundingClientRect();
  const scaleX = videoPlayer.videoWidth / videoPlayer.offsetWidth;
  const scaleY = videoPlayer.videoHeight / videoPlayer.offsetHeight;

  const x = event.clientX - videoRect.left;
  const y = event.clientY - videoRect.top;

  coordinates.push({
    x: x * scaleX,
    y: y * scaleY,
  });
  labels.push(label);

  const shape = document.createElement("div");
  shape.className = shapeType;
  shape.style.left = `${x}px`;
  shape.style.top = `${y}px`;
  labelsContainer.appendChild(shape);
}

let activeLabel = null;

const updateLabelButtonUI = () => {
  if (activeLabel === "positive") {
    document.body.style.cursor = "crosshair";
    addPositiveLabelButton.style.color = "green";
    addNegativeLabelButton.style.color = "white";
    isVideoPlayable = false;
  } else if (activeLabel === "negative") {
    document.body.style.cursor = "not-allowed";
    addPositiveLabelButton.style.color = "white";
    addNegativeLabelButton.style.color = "red";
    isVideoPlayable = false;
  } else {
    document.body.style.cursor = "default";
    addPositiveLabelButton.style.color = "white";
    addNegativeLabelButton.style.color = "white";
    isVideoPlayable = true;
  }
};

addPositiveLabelButton.addEventListener("click", () => {
  activeLabel = activeLabel === "positive" ? null : "positive";
  updateLabelButtonUI();
});

addNegativeLabelButton.addEventListener("click", () => {
  activeLabel = activeLabel === "negative" ? null : "negative";
  updateLabelButtonUI();
});

videoPlayer.addEventListener("click", (event) => {
  if (!isVideoPlayable) {
    event.preventDefault();
    if (activeLabel === "positive") {
      addPoint(event, "circle", 1);
    } else {
      addPoint(event, "cross", 0);
    }
  }
});

// image preview
const imageInput = document.getElementById("input-img");
imageInput.addEventListener("change", (event) => {
  const imgPreview = document.getElementById("img-preview");
  const file = event.target.files[0];
  const reader = new FileReader();

  reader.onload = function () {
    imgPreview.src = reader.result;
    imgPreview.style.display = "block";
  };

  if (file) {
    reader.readAsDataURL(file);
  }
});

// VIDEO UPLOAD to R2 bucket + WEBSOCKET CONNECTION
//

// let ws;
// let videoKey = localStorage.getItem("videoKey");
// if (videoKey) {
//   const connToWs = async () => {
//     await connectToWebSocket(videoKey);
//   };

//   videoInputUpload.disabled = true;
//   connToWs();
// }

// async function displayVideo(videoKey) {
//   const presignedGetUrl = BACKEND_URL + "/presigned-url/get?key=" + videoKey;
//   try {
//     const presignedGetResponse = await fetch(presignedGetUrl, {
//       method: "POST",
//     });
//     const { presignedUrl: getUrl } = await presignedGetResponse.json();

//     // show video in the video player
//     videoPlayer.src = getUrl;
//     videoPlayer.style.display = "block";
//   } catch (error) {
//     console.error("Error fetching presigned GET URL:", error);
//   }
// }

// const videoPreviewMessage = document.getElementById("video-preview-msg");
// connect to websocket for event-driven workflow
// async function connectToWebSocket(videoKey) {
//   return new Promise((resolve, reject) => {
//     ws = new WebSocket(BACKEND_WS_URL);

//     ws.onopen = () => {
//       console.log("WebSocket connection established");
//       ws.send(JSON.stringify({ videoKey: videoKey }));
//       resolve();
//     };

//     ws.onerror = (error) => {
//       console.error("WebSocket error:", error);
//       reject(error);
//     };

//     ws.onmessage = async (event) => {
//       const message = JSON.parse(event.data);
//       console.log("Received message from server:", message);

//       if (message.status === "completed") {
//         videoPreviewMessage.hidden = true;
//         videoInferenceContainer.hidden = false;
//         await displayVideo(message.videoKey);
//         inferenceVideoButtonElement.disabled = false;
//         addPositiveLabelButton.disabled = false;
//         addNegativeLabelButton.disabled = false;
//       }
//     };
//   });
// }

//async function uploadVideoAndConnectToWebsocket(videoFile) {
//  videoPreviewMessage.hidden = false;
//
//  // upload video to r2 bucket with presigned url
//  const presignedPutUrl = BACKEND_URL + "/presigned-url/put";
//  const presignedPutResponse = await fetch(presignedPutUrl, {
//    method: "POST",
//  });
//
//  const { presignedUrl: uploadUrl, key: newVideoKey } =
//    await presignedPutResponse.json();
//
//  await fetch(uploadUrl, {
//    method: "PUT",
//    body: videoFile,
//  });
//
//  videoKey = newVideoKey;
//
//  localStorage.setItem("videoKey", videoKey);
//
//  await connectToWebSocket(videoKey);
//}

// INFERENCE
//
inferenceVideoButtonElement.addEventListener("click", async () => {
  console.log("GIVE ME CREDITS FOR INFERENCE");

  if (videoPlayer.duration > 5) {
    alert("Video must be 5 seconds or shorter. Please trim the video.");
    return;
  }

  const imageFile = imageInput.files[0];
  const videoFile = await getVideoFile();

  if (!imageFile) {
    alert("Please select an image.");
    return;
  }

  if (!videoFile) {
    alert("Please select a video.");
    return;
  }

  if (coordinates.length === 0) {
    alert("Please add points for segmentation.");
    return;
  }

  const formData = new FormData();

  formData.append("image", imageFile);
  formData.append("video", videoFile);
  formData.append(
    "segmentationData",
    JSON.stringify({
      coordinates: coordinates,
      labels: labels,
    }),
  );

  loadingSpinnerContainer.style.display = "flex";
  inferenceVideoButtonElement.hidden = true;
  addPositiveLabelButton.disabled = true;
  addNegativeLabelButton.disabled = true;
  imageInput.disabled = true;
  videoInputUpload.disabled = true;

  try {
    const response = await fetch(BACKEND_URL + "/video/local-inference", {
      method: "POST",
      body: formData,
      headers: {
        Accept: "video/mp4",
      },
    });

    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(
        errorData.error || `HTTP error! status: ${response.status}`,
      );
    }

    coordinates = [];
    labels = [];
    labelsContainer.innerHTML = "";
    addPositiveLabelButton.disabled = false;
    addNegativeLabelButton.disabled = false;
    imageInput.disabled = false;
    videoInputUpload.disabled = false;

    // download video directly in the browser after inference
    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = generateRandomFilename();
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    a.remove();
  } catch (error) {
    console.error("Error:", error);
  } finally {
    loadingSpinnerContainer.style.display = "none";
    inferenceVideoButtonElement.hidden = false;
  }
});
// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
