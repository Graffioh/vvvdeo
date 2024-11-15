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
  });

  addNegativeLabelButton.addEventListener("click", () => {
    isVideoPlayable = false;
    isPositiveLabel = false;
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

  const inferenceVideoButtonElement = document.getElementById(
    "inference-video-btn",
  );

  inferenceVideoButtonElement.addEventListener("click", () => {
    fetch(backendUrl + "/inference-video", {
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
        const segmentedImages = data.segmented_image_paths;
        segmentedImages.forEach((imgName) => {
          const img = new Image();
          img.src = "./sam2seg/" + imgName;
          img.alt = imgName;
          imageContainer.appendChild(img);
        });
      })
      .catch((error) => console.error("Error:", error));
  });
  // +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
});
