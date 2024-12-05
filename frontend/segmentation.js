import { fetchFile, toBlobURL } from "@ffmpeg/util";
import { FFmpeg } from "@ffmpeg/ffmpeg";

const backendUrl = import.meta.env.VITE_BACKEND_URL;
const backendWsUrl = import.meta.env.VITE_BACKEND_WS_URL;

document.addEventListener("DOMContentLoaded", () => {
  const inferenceVideoButtonElement = document.getElementById(
    "inference-video-btn",
  );

  // VIDEO SEGMENTATION
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
  const videoPlayer = document.getElementById("video-player");
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

  // VIDEO UPLOAD + WEBSOCKET CONNECTION
  //
  const videoInferenceContainer = document.getElementById(
    "inference-container",
  );
  const videoInputUpload = document.getElementById("input-video");

  async function displayVideo(videoKey) {
    const presignedGetUrl = backendUrl + "/presigned-get-url?key=" + videoKey;
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

  let ws;
  let videoKey = localStorage.getItem("videoKey");
  if (videoKey) {
    const connToWs = async () => {
      await connectToWebSocket(videoKey);
    };

    videoInputUpload.disabled = true;
    connToWs();
  }

  const videoPreviewMessage = document.getElementById("video-preview-msg");
  // connect to websocket for event-driven workflow
  async function connectToWebSocket(videoKey) {
    return new Promise((resolve, reject) => {
      ws = new WebSocket(backendWsUrl);

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
    const presignedPutUrl = backendUrl + "/presigned-put-url";
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

  // ffmpeg wasm trimming
  let trimmedVideoFile = null;
  let ffmpeg = null;
  const baseURL = "https://unpkg.com/@ffmpeg/core@0.12.6/dist/esm";
  const trim = async (file, startTrim, endTrim) => {
    const ffmpegMessage = document.getElementById("ffmpeg-message");
    if (ffmpeg === null) {
      ffmpeg = new FFmpeg();
      ffmpeg.on("log", ({ message }) => {
        console.log(message);
      });
      ffmpeg.on("progress", ({ progress }) => {
        ffmpegMessage.innerHTML = `${progress * 100} %`;
      });
      await ffmpeg.load({
        coreURL: await toBlobURL(
          `${baseURL}/ffmpeg-core.js`,
          "text/javascript",
        ),
        wasmURL: await toBlobURL(
          `${baseURL}/ffmpeg-core.wasm`,
          "application/wasm",
        ),
      });
    }

    const { name } = file;
    await ffmpeg.writeFile(name, await fetchFile(file));
    ffmpegMessage.innerHTML = "Start trimming...";
    await ffmpeg.exec([
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

  const trimInputs = document.getElementById("inputs-trim");
  const trimInputsContainer = document.getElementById("trim-container");
  const trimButton = document.getElementById("trim-button");

  videoInputUpload.addEventListener("change", () => {
    const videoFile = videoInputUpload.files[0];
    const videoURL = URL.createObjectURL(videoFile);

    videoPlayer.src = videoURL;
    videoPlayer.style.display = "block";
    trimInputsContainer.style.display = "flex";
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
    return `${hrs}:${mins}:${secs}`;
  }

  const startTrimInput = document.getElementById("start-trim-input");
  const endTrimInput = document.getElementById("end-trim-input");

  const startTrimBtn = document.getElementById("start-trim-btn");
  const endTrimBtn = document.getElementById("end-trim-btn");
  startTrimBtn.addEventListener("click", () => {
    startTrimInput.value = secondsToTime(videoPlayer.currentTime);
  });
  endTrimBtn.addEventListener("click", () => {
    if (!startTrimInput.value) {
      alert("Please select first the starting time.");
      return;
    }

    if (videoPlayer.currentTime < timeToSeconds(startTrimInput.value)) {
      alert("Please select a valid starting and ending time");
      return;
    }

    endTrimInput.value = secondsToTime(videoPlayer.currentTime);
  });

  trimButton.addEventListener("click", async () => {
    const startTrimValue = startTrimInput.value;
    const endTrimValue = endTrimInput.value;

    const startTrimSeconds = timeToSeconds(startTrimValue);
    const endTrimSeconds = timeToSeconds(endTrimValue);

    if (endTrimSeconds - startTrimSeconds > 10) {
      alert("Video needs to be maximum 10 seconds long.");
      return;
    }

    const videoFile = videoInputUpload.files[0];
    await trim(videoFile, startTrimValue, endTrimValue);

    // trimInputsContainer.style.display = "none";
    // videoInferenceContainer.style.display = "block";
    // uploadVideoAndConnectToWebsocket(trimmedVideoFile);
  });

  // INFERENCE
  //
  const spinner = document.getElementById("loading-spinner");
  const loadingText = document.getElementById("loading-text");

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

    spinner.style.display = "block";
    loadingText.style.display = "block";
    inferenceVideoButtonElement.hidden = true;
    addPositiveLabelButton.disabled = true;
    addNegativeLabelButton.disabled = true;
    imageInput.disabled = true;
    videoInputUpload.disabled = true;

    try {
      const response = await fetch(backendUrl + "/inference-video", {
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
      a.download = "processed_video.mp4";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();

      localStorage.removeItem("videoKey");
    } catch (error) {
      console.error("Error:", error);
    } finally {
      spinner.style.display = "none";
      loadingText.style.display = "none";
      inferenceVideoButtonElement.hidden = false;
    }
  });
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
});
