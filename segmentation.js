import { VideoPlayer } from "./videoPlayer.js";

const backendUrl = "http://localhost:8080";

document.addEventListener("DOMContentLoaded", () => {
  // VIDEO SEGMENTATION
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

  const videoElement = document.getElementById("shaka-player-video");
  const coordinates = [];
  const labels = [];
  const shapesContainer = document.getElementById("shapes-container");

  const addPositiveLabelButton = document.getElementById("seg-add");
  const addNegativeLabelButton = document.getElementById("seg-excl");
  let isVideoPlayable = true;
  let isPositiveLabel = true;

  function addPoint(event, shapeType, label) {
    const videoRect = videoElement.getBoundingClientRect();
    const scaleX = videoElement.videoWidth / videoElement.offsetWidth;
    const scaleY = videoElement.videoHeight / videoElement.offsetHeight;

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

  videoElement.addEventListener("click", (event) => {
    if (!isVideoPlayable) {
      event.preventDefault();
      if (isPositiveLabel) {
        addPoint(event, "circle", 1);
      } else {
        addPoint(event, "square", 0);
      }
    }
  });

  /*
  const inferenceFrameButtonElement = document.getElementById(
    "inference-frame-btn",
  );

  inferenceFrameButtonElement.addEventListener("click", () => {
    fetch(backendUrl + "/inference-frame", {
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
        segmentedFrame.src = "./sam2seg/" + data.segmented_image_path;
        segmentedFrame.style.display = "block";
      })
      .catch((error) => console.error("Error:", error));
  });
  */

  const fileInput = document.getElementById("input-img");

  // image preview
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

  // video upload
  let videoName = "";
  const videoInputUpload = document.getElementById("input-video");
  videoInputUpload.addEventListener("change", async (event) => {
    event.preventDefault();
    const videoFile = videoInputUpload.files[0];
    videoName = videoFile.name;

    // R2 bucket upload
    const url = "http://localhost:8080/presigned-put-url";
    const response = await fetch(url, { method: "POST" });

    const { presignedUrl: uploadUrl } = await response.json();

    await fetch(uploadUrl, {
      method: "PUT",
      body: videoFile,
    });

    /*

    const formData = new FormData();
    formData.append("video", videoFile);
    formData.append("video_name", videoName);

    try {
      const response = await fetch(backendUrl + "/uploadvideo", {
        method: "POST",
        body: formData,
      });

      if (response.ok) {
        alert("Video uploaded successfully");
      } else {
        alert("Video upload failed");
      }
    } catch (error) {
      console.error("Error:", error);
    }

    const videoPlayer = new VideoPlayer();
    await videoPlayer.loadManifest("http://localhost:8080/zawarudo/.mp4");
    videoPlayer.setVideoPlayerVisible();
    */
  });

  // inference
  const inferenceVideoButtonElement = document.getElementById(
    "inference-video-btn",
  );

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
