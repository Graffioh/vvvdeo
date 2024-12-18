import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { FFmpeg } from "@ffmpeg/ffmpeg";

const BACKEND_URL = import.meta.env.VITE_BACKEND_URL;
const BACKEND_WS_URL = import.meta.env.VITE_BACKEND_WS_URL;

const videoPlayer = document.getElementById("video-player");
const videoInputUpload = document.getElementById("input-video");
const videoPlayerContainer = document.getElementById("video-player-container");
const loadingSpinnerContainer = document.getElementById(
  "loading-spinner-container",
);
const ffmpegMessage = document.getElementById("ffmpeg-message");

// ffmpeg wasm trimming
const BASE_FFMPEG_WASM_URL = "https://unpkg.com/@ffmpeg/core@0.12.6/dist/esm";

let trimmedVideoFile = null;
let ffmpeg = null;
const trim = async (file, startTrim, endTrim) => {
  if (ffmpeg === null) {
    ffmpeg = new FFmpeg();
    ffmpeg.on("log", ({ message }) => {
      console.log(message);
    });
    ffmpeg.on("progress", ({ progress, time }) => {
      ffmpegMessage.innerHTML = `${progress * 100} % (transcoded time: ${time / 1000000} s)`;
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
  ffmpegMessage.hidden = true;
};

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

const convertStreamToFile = async () => {
  try {
    const videoUrl = videoPlayer.src;
    const response = await fetch(videoUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch video: ${response.statusText}`);
    }
    const videoBlob = await response.blob();
    const videoFile = new File([videoBlob], "streamedVideo.mp4", {
      type: videoBlob.type,
    });
    return videoFile;
  } catch (error) {
    console.error("Error converting video stream to file:", error);
    alert("Failed to convert video stream to file.");
    return null;
  }
};

var ffmpegSSE = new EventSource(BACKEND_URL + "/ffmpeg-events");
ffmpegSSE.onmessage = function (event) {
  console.log("sse message received");
  ffmpegMessage.innerHTML = event.data;
};
ffmpegSSE.onerror = function () {
  setTimeout(() => (ffmpegMessage.innerHTML = ""), 1000);
};

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

showTrimButton.addEventListener("click", () => {
  ffmpegInputsContainer.style.display = "flex";

  speedupButton.style.display = "none";
  trimButtonFast.style.display = "block";
  speedupFactorContainer.style.display = "none";
});

showSpeedupButton.addEventListener("click", () => {
  ffmpegInputsContainer.style.display = "flex";

  speedupButton.style.display = "block";
  trimButtonFast.style.display = "none";
  speedupFactorContainer.style.display = "block";
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

trimButtonFast.addEventListener("click", async () => {
  const startTrimValue = startTimestampInput.value;
  const endTrimValue = endTimestampInput.value;

  const startTrimSeconds = timeToSeconds(startTrimValue);
  const endTrimSeconds = timeToSeconds(endTrimValue);

  //if (endTrimSeconds - startTrimSeconds > 10) {
  //  alert("Video needs to be maximum 10 seconds long.");
  //  return;
  //}

  if (videoInputUpload.files[0]) {
    await trim(videoInputUpload.files[0], startTrimValue, endTrimValue);
  } else {
    // for youtube downloaded video (wip)
    const videoStreamFile = await convertStreamToFile();
    await trim(videoStreamFile, startTrimValue, endTrimValue);
  }

  console.log("GIVE ME CREDITS FOR INFERENCE");

  startTimestampInput.value = null;
  endTimestampInput.value = null;

  // trimInputsContainer.style.display = "none";
  // videoInferenceContainer.style.display = "block";
  // uploadVideoAndConnectToWebsocket(trimmedVideoFile);
});

speedupButton.addEventListener("click", async () => {
  const startTrimValue =
    startTimestampInput.value === "00:00:00.000"
      ? "00:00:00.100"
      : startTimestampInput.value;

  const endTrimValue = endTimestampInput.value;

  const videoInputFile = videoInputUpload.files[0];

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
const shapesContainer = document.getElementById("shapes-container");

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
  shapesContainer.appendChild(shape);
}

let activeLabel = null;

const updateLabelButtonUI = () => {
  if (activeLabel === "positive") {
    document.body.style.cursor = "crosshair";
    addPositiveLabelButton.style.color = "black";
    addNegativeLabelButton.style.color = "white";
    isVideoPlayable = false;
  } else if (activeLabel === "negative") {
    document.body.style.cursor = "not-allowed";
    addPositiveLabelButton.style.color = "white";
    addNegativeLabelButton.style.color = "black";
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
      addPoint(event, "square", 0);
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
const videoInferenceContainer = document.getElementById("inference-container");

let ws;
let videoKey = localStorage.getItem("videoKey");
if (videoKey) {
  const connToWs = async () => {
    await connectToWebSocket(videoKey);
  };

  videoInputUpload.disabled = true;
  connToWs();
}

async function displayVideo(videoKey) {
  const presignedGetUrl = BACKEND_URL + "/presigned-url/get?key=" + videoKey;
  try {
    const presignedGetResponse = await fetch(presignedGetUrl, {
      method: "POST",
    });
    const { presignedUrl: getUrl } = await presignedGetResponse.json();

    // show video in the video player
    videoPlayer.src = getUrl;
    videoPlayer.style.display = "block";
  } catch (error) {
    console.error("Error fetching presigned GET URL:", error);
  }
}

const videoPreviewMessage = document.getElementById("video-preview-msg");
// connect to websocket for event-driven workflow
async function connectToWebSocket(videoKey) {
  return new Promise((resolve, reject) => {
    ws = new WebSocket(BACKEND_WS_URL);

    ws.onopen = () => {
      console.log("WebSocket connection established");
      ws.send(JSON.stringify({ videoKey: videoKey }));
      resolve();
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      reject(error);
    };

    ws.onmessage = async (event) => {
      const message = JSON.parse(event.data);
      console.log("Received message from server:", message);

      if (message.status === "completed") {
        videoPreviewMessage.hidden = true;
        videoInferenceContainer.hidden = false;
        await displayVideo(message.videoKey);
        inferenceVideoButtonElement.disabled = false;
        addPositiveLabelButton.disabled = false;
        addNegativeLabelButton.disabled = false;
      }
    };
  });
}

async function uploadVideoAndConnectToWebsocket(videoFile) {
  videoPreviewMessage.hidden = false;

  // upload video to r2 bucket with presigned url
  const presignedPutUrl = BACKEND_URL + "/presigned-url/put";
  const presignedPutResponse = await fetch(presignedPutUrl, {
    method: "POST",
  });

  const { presignedUrl: uploadUrl, key: newVideoKey } =
    await presignedPutResponse.json();

  await fetch(uploadUrl, {
    method: "PUT",
    body: videoFile,
  });

  videoKey = newVideoKey;

  localStorage.setItem("videoKey", videoKey);

  await connectToWebSocket(videoKey);
}

// INFERENCE
//
inferenceVideoButtonElement.addEventListener("click", async () => {
  const imageFile = imageInput.files[0];

  if (!imageFile) {
    alert("Please select an image.");
    return;
  }

  if (coordinates.length === 0) {
    alert("Please add points for segmentation.");
    return;
  }

  const formData = new FormData();

  formData.append("image", imageFile);
  formData.append("videoKey", videoKey);
  formData.append(
    "segmentationData",
    JSON.stringify({
      coordinates: coordinates,
      labels: labels,
    }),
  );

  loadingSpinnerContainer.style.display = "block";
  inferenceVideoButtonElement.hidden = true;
  addPositiveLabelButton.disabled = true;
  addNegativeLabelButton.disabled = true;
  imageInput.disabled = true;
  videoInputUpload.disabled = true;

  try {
    const response = await fetch(BACKEND_URL + "/video/inference", {
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
    shapesContainer.innerHTML = "";
    addPositiveLabelButton.disabled = false;
    addNegativeLabelButton.disabled = false;
    imageInput.disabled = false;
    videoInputUpload.disabled = false;

    // download video directly in the browser after inference
    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "crafted_vvvdeo.mp4";
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    a.remove();

    localStorage.removeItem("videoKey");
  } catch (error) {
    console.error("Error:", error);
  } finally {
    loadingSpinnerContainer.style.display = "none";
    inferenceVideoButtonElement.hidden = false;
  }
});
// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
