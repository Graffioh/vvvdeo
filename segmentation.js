const backendUrl = "http://localhost:8080";

document.addEventListener("DOMContentLoaded", () => {
  // VIDEO SEGMENTATION
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
  const videoPlayer = document.getElementById("video-player");
  const coordinates = [];
  const labels = [];
  const shapesContainer = document.getElementById("shapes-container");

  const addPositiveLabelButton = document.getElementById("seg-add");
  const addNegativeLabelButton = document.getElementById("seg-excl");
  let isVideoPlayable = true;
  let isPositiveLabel = true;

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

  addPositiveLabelButton.addEventListener("click", () => {
    isVideoPlayable = false;
    isPositiveLabel = true;
    document.body.style.cursor = "crosshair";
    addPositiveLabelButton.style.color = "black";
    addNegativeLabelButton.style.color = "white";
  });

  addNegativeLabelButton.addEventListener("click", () => {
    isVideoPlayable = false;
    isPositiveLabel = false;
    document.body.style.cursor = "not-allowed";
    addPositiveLabelButton.style.color = "white";
    addNegativeLabelButton.style.color = "black";
  });

  videoPlayer.addEventListener("click", (event) => {
    if (!isVideoPlayable) {
      event.preventDefault();
      if (isPositiveLabel) {
        addPoint(event, "circle", 1);
      } else {
        addPoint(event, "square", 0);
      }
    }
  });

  // image preview
  const fileInput = document.getElementById("input-img");

  fileInput.addEventListener("change", (event) => {
    const preview = document.getElementById("preview");
    const file = event.target.files[0];
    const reader = new FileReader();

    reader.onload = function () {
      preview.src = reader.result;
      preview.style.display = "block";
    };

    if (file) {
      reader.readAsDataURL(file);
    }
  });

  const inferenceVideoButtonElement = document.getElementById(
    "inference-video-btn",
  );

  // VIDEO UPLOAD + WEBSOCKET CONNECTION
  //
  let videoName = "";
  const videoInputUpload = document.getElementById("input-video");
  let ws;

  let wsKey = localStorage.getItem("videoKey");

  if (wsKey) {
    const connToWs = async () => {
      await connectWebSocket(wsKey);
    };

    connToWs();
  }

  async function connectWebSocket(videoKey) {
    return new Promise((resolve, reject) => {
      ws = new WebSocket("ws://localhost:8080/ws");

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
          await displayVideo(message.videoKey);
          localStorage.removeItem("videoKey");
        }
      };
    });
  }

  async function displayVideo(videoKey) {
    const presignedGetUrl =
      "http://localhost:8080/presigned-get-url?key=" + videoKey;

    try {
      const presignedGetResponse = await fetch(presignedGetUrl, {
        method: "POST",
      });

      const { presignedUrl: getUrl } = await presignedGetResponse.json();

      // Show video in the video player
      videoPlayer.src = getUrl;
      videoPlayer.style.display = "block";

      // Enable run inference button
      inferenceVideoButtonElement.disabled = false;
    } catch (error) {
      console.error("Error fetching presigned GET URL:", error);
    }
  }

  async function uploadVideoAndConnectToWebsocket(videoFile) {
    // upload video to r2 bucket with presigned url
    const presignedPutUrl = "http://localhost:8080/presigned-put-url";
    const presignedPutResponse = await fetch(presignedPutUrl, {
      method: "POST",
    });

    const { presignedUrl: uploadUrl, key: videoKey } =
      await presignedPutResponse.json();

    await fetch(uploadUrl, {
      method: "PUT",
      body: videoFile,
    });

    localStorage.setItem("videoKey", videoKey);

    await connectWebSocket(videoKey);
  }

  videoInputUpload.addEventListener("change", async (event) => {
    event.preventDefault();
    const videoFile = videoInputUpload.files[0];
    videoName = videoFile.name;

    uploadVideoAndConnectToWebsocket(videoFile);
  });

  // INFERENCE
  //
  const spinner = document.getElementById("loading-spinner");
  const loadingText = document.getElementById("loading-text");

  inferenceVideoButtonElement.addEventListener("click", async () => {
    const imageFile = fileInput.files[0];
    const formData = new FormData();

    formData.append("image", imageFile);
    formData.append("video_name", videoName);
    formData.append(
      "data",
      JSON.stringify({
        coordinates: coordinates,
        labels: labels,
      }),
    );

    spinner.style.display = "block";
    loadingText.style.display = "block";
    inferenceVideoButtonElement.hidden = true;

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

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "processed_video.mp4";
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();

      inferenceVideoButtonElement.hidden = false;
    } catch (error) {
      console.error("Error:", error);
    } finally {
      spinner.style.display = "none";
      loadingText.style.display = "none";
    }
  });
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
});
