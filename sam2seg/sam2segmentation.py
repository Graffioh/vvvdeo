'''
import os
# if using Apple MPS, fall back to CPU for unsupported ops
os.environ["PYTORCH_ENABLE_MPS_FALLBACK"] = "1"
import numpy as np
import torch
import matplotlib.pyplot as plt
from PIL import Image
from sam2.build_sam import build_sam2_video_predictor


# select the device for computation
if torch.cuda.is_available():
    device = torch.device("cuda")
elif torch.backends.mps.is_available():
    device = torch.device("mps")
else:
    device = torch.device("cpu")
print(f"using device: {device}")

if device.type == "cuda":
    # use bfloat16 for the entire notebook
    torch.autocast("cuda", dtype=torch.bfloat16).__enter__()
    # turn on tfloat32 for Ampere GPUs (https://pytorch.org/docs/stable/notes/cuda.html#tensorfloat-32-tf32-on-ampere-devices)
    if torch.cuda.get_device_properties(0).major >= 8:
        torch.backends.cuda.matmul.allow_tf32 = True
        torch.backends.cudnn.allow_tf32 = True
elif device.type == "mps":
    print(
        "\nSupport for MPS devices is preliminary. SAM 2 is trained with CUDA and might "
        "give numerically different outputs and sometimes degraded performance on MPS. "
        "See e.g. https://github.com/pytorch/pytorch/issues/84936 for a discussion."
    )


sam2_checkpoint = "./segment-anything-2/checkpoints/sam2.1_hiera_tiny.pt"
model_cfg = "./configs/sam2.1/sam2.1_hiera_t.yaml"

predictor = build_sam2_video_predictor(model_cfg, sam2_checkpoint, device=device)

def show_mask(mask, ax, obj_id=None, random_color=False):
    if random_color:
        color = np.concatenate([np.random.random(3), np.array([0.6])], axis=0)
    else:
        cmap = plt.get_cmap("tab10")
        cmap_idx = 0 if obj_id is None else obj_id
        color = np.array([*cmap(cmap_idx)[:3], 0.6])
    h, w = mask.shape[-2:]
    mask_image = mask.reshape(h, w, 1) * color.reshape(1, 1, -1)
    ax.imshow(mask_image)


def show_points(coords, labels, ax, marker_size=200):
    pos_points = coords[labels==1]
    neg_points = coords[labels==0]
    ax.scatter(pos_points[:, 0], pos_points[:, 1], color='green', marker='*', s=marker_size, edgecolor='white', linewidth=1.25)
    ax.scatter(neg_points[:, 0], neg_points[:, 1], color='red', marker='*', s=marker_size, edgecolor='white', linewidth=1.25)


def show_box(box, ax):
    x0, y0 = box[0], box[1]
    w, h = box[2] - box[0], box[3] - box[1]
    ax.add_patch(plt.Rectangle((x0, y0), w, h, edgecolor='green', facecolor=(0, 0, 0, 0), lw=2))

# `video_dir` a directory of JPEG frames with filenames like `<frame_index>.jpg`
video_dir = "./frames"

# scan all the JPEG frame names in this directory
frame_names = [
    p for p in os.listdir(video_dir)
    if os.path.splitext(p)[-1] in [".jpg", ".jpeg", ".JPG", ".JPEG"]
]
frame_names.sort(key=lambda p: int(os.path.splitext(p)[0]))

# take a look the first video frame
frame_idx = 0
plt.figure(figsize=(9, 6))
plt.title(f"frame {frame_idx}")
plt.imshow(Image.open(os.path.join(video_dir, frame_names[frame_idx])))

inference_state = predictor.init_state(video_path=video_dir)

predictor.reset_state(inference_state)

ann_frame_idx = 0  # the frame index we interact with
ann_obj_id = 1  # give a unique id to each object we interact with (it can be any integers)

# Let's add a positive click at (x, y) = (210, 350) to get started
points = np.array([[508, 432], [709, 574], [551, 598], [1013, 347], [279, 243], [225, 617]], dtype=np.float32)
# for labels, `1` means positive click and `0` means negative click
labels = np.array([1, 1, 1, 0, 0, 0], np.int32)
_, out_obj_ids, out_mask_logits = predictor.add_new_points_or_box(
    inference_state=inference_state,
    frame_idx=ann_frame_idx,
    obj_id=ann_obj_id,
    points=points,
    labels=labels,
)

# show the results on the current (interacted) frame
plt.figure(figsize=(9, 6))
plt.title(f"frame {ann_frame_idx}")
plt.imshow(Image.open(os.path.join(video_dir, frame_names[ann_frame_idx])))
show_points(points, labels, plt.gca())
show_mask((out_mask_logits[0] > 0.0).cpu().numpy(), plt.gca(), obj_id=out_obj_ids[0])


# run propagation throughout the video and collect the results in a dict
video_segments = {}  # video_segments contains the per-frame segmentation results
for out_frame_idx, out_obj_ids, out_mask_logits in predictor.propagate_in_video(inference_state):
    video_segments[out_frame_idx] = {
        out_obj_id: (out_mask_logits[i] > 0.0).cpu().numpy()
        for i, out_obj_id in enumerate(out_obj_ids)
    }

# render the segmentation results every few frames
vis_frame_stride = 30
plt.close("all")
for out_frame_idx in range(0, len(frame_names), vis_frame_stride):
    plt.figure(figsize=(6, 4))
    plt.title(f"frame {out_frame_idx}")
    plt.imshow(Image.open(os.path.join(video_dir, frame_names[out_frame_idx])))
    for out_obj_id, out_mask in video_segments[out_frame_idx].items():
        show_mask(out_mask, plt.gca(), obj_id=out_obj_id)

plt.show()
'''

import os
import json
import numpy as np
import torch
from flask import Flask, request, jsonify, send_file
from sam2.build_sam import build_sam2_video_predictor
from PIL import Image, ImageOps, ImageFilter
import time
import cv2
import supervision as sv

def clear_directory(directory_path):
    for filename in os.listdir(directory_path):
        file_path = os.path.join(directory_path, filename)
        if os.path.isfile(file_path):
            os.remove(file_path)
        elif os.path.isdir(file_path):
            shutil.rmtree(file_path)

clear_directory("./static/")

app = Flask(__name__)

device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
sam2_checkpoint = "./src/sam-2/checkpoints/sam2.1_hiera_small.pt"
model_cfg = "./configs/sam2.1/sam2.1_hiera_s.yaml"
predictor = build_sam2_video_predictor(model_cfg, sam2_checkpoint, device=device)

video_dir = "./frames"

@app.route("/predict", methods=["POST"])
def predict():
    inference_state = predictor.init_state(video_path=video_dir)
    predictor.reset_state(inference_state)

    try:
        data = request.get_json()
        points = data.get("coordinates", [])
        labels = data.get("labels", [])

        if not points:
            return jsonify({"error": "Missing coordinates"}), 400

        try:
            points = np.array([[p['x'], p['y']] for p in points], dtype=np.float32)
            labels = np.array([l for l in labels], dtype=np.int32)  
        except Exception as e:
            return jsonify({"error": f"Error processing points: {str(e)}"}), 500

        ann_frame_idx = 0  
        ann_obj_id = 1

        _, out_obj_ids, out_mask_logits = predictor.add_new_points_or_box(
            inference_state=inference_state,
            frame_idx=ann_frame_idx,
            obj_id=ann_obj_id,
            points=points,
            labels=labels,
        )

        mask_data = (out_mask_logits[0] > 0.0).cpu().numpy()
        color = np.concatenate([np.random.random(3), np.array([0.6])], axis=0)  
        h, w = mask_data.shape[-2:]
        mask_image = mask_data.reshape(h, w, 1) * color.reshape(1, 1, -1)
        mask_image_rgb = (mask_image[..., :3] * 255).astype(np.uint8)  
        segmented_image = Image.fromarray(mask_image_rgb, 'RGB')

        output_dir = "./static"
        os.makedirs(output_dir, exist_ok=True)
        timestamp = str(time.time_ns())
        segmented_image_path = os.path.join(output_dir, f"segmented_frame_{timestamp}.png")
        segmented_image.save(segmented_image_path)

        return jsonify({
            "frame_idx": ann_frame_idx,
            "obj_id": int(out_obj_ids[0]),  
            "status": "success",
            "segmented_image_path": segmented_image_path
        })

    except Exception as e:
        return jsonify({"error": str(e), "status": "error"}), 500

@app.route("/predict-frames", methods=["POST"])
def predict_frames():
    inference_state = predictor.init_state(video_path=video_dir)
    predictor.reset_state(inference_state)
    try:
        data = request.get_json()
        points = data.get("coordinates", [])
        labels = data.get("labels", [])
        if not points:
            return jsonify({"error": "Missing coordinates"}), 400
        try:
            points = np.array([[p['x'], p['y']] for p in points], dtype=np.float32)
            labels = np.array([l for l in labels], dtype=np.int32)  
        except Exception as e:
            return jsonify({"error": f"Error processing points: {str(e)}"}), 500

        ann_frame_idx = 0  
        ann_obj_id = 1
        _, out_obj_ids, out_mask_logits = predictor.add_new_points_or_box(
            inference_state=inference_state,
            frame_idx=ann_frame_idx,
            obj_id=ann_obj_id,
            points=points,
            labels=labels,
        )

        colors = ['#FF1493', '#00BFFF', '#FF6347', '#FFD700']
        mask_annotator = sv.MaskAnnotator(
            color=sv.ColorPalette.from_hex(colors),
            color_lookup=sv.ColorLookup.TRACK,
            opacity=0
        )

        video_info = sv.VideoInfo.from_video_path("./jojorun.mp4")
        frames_paths = sorted(sv.list_files_with_extensions(
            directory="./frames/", 
            extensions=["jpg"]))

        overlay_img = cv2.imread("./img/X_logo.png", cv2.IMREAD_UNCHANGED)

        if len(overlay_img.shape) == 2:  
            overlay_img = cv2.cvtColor(overlay_img, cv2.COLOR_GRAY2RGB)
        
        if overlay_img is not None:
            if len(overlay_img.shape) == 3 and overlay_img.shape[2] == 3:
                alpha_channel = np.ones(overlay_img.shape[:2], dtype=overlay_img.dtype) * 255
                overlay_img = cv2.merge([overlay_img, alpha_channel])
            elif len(overlay_img.shape) != 3 or overlay_img.shape[2] != 4:
                raise ValueError("Overlay image must be in RGB or RGBA format")
        else:
            raise ValueError("Could not load overlay image")

        with sv.VideoSink("./static/video_result.mp4", video_info=video_info) as sink:
            for frame_idx, object_ids, mask_logits in predictor.propagate_in_video(inference_state):
                if not os.path.exists(frames_paths[frame_idx]):
                    print(f"Warning: Frame '{frames_paths}' does not exist.")
                    continue

                frame = cv2.imread(frames_paths[frame_idx])
                if frame is None:
                    print(f"Warning: Could not read frame '{frames_paths}'.")
                    continue

                masks = (mask_logits > 0.0).cpu().numpy()
                N, X, H, W = masks.shape
                masks = masks.reshape(N * X, H, W)
                
                detections = sv.Detections(
                    xyxy=sv.mask_to_xyxy(masks=masks),
                    mask=masks,
                    tracker_id=np.array(object_ids)
                )

                result_frame = frame.copy()

                for i, mask in enumerate(masks):
                    if not mask.any():  
                        continue

                    y_indices, x_indices = np.where(mask)
                    if len(y_indices) == 0 or len(x_indices) == 0:
                        continue

                    y_min, y_max = np.min(y_indices), np.max(y_indices)
                    x_min, x_max = np.min(x_indices), np.max(x_indices)
                    
                    mask_height = y_max - y_min + 1
                    mask_width = x_max - x_min + 1

                    resized_overlay = cv2.resize(overlay_img, (mask_width, mask_height))
                    
                    current_mask = mask[y_min:y_max+1, x_min:x_max+1]
                    
                    overlay_color = resized_overlay[:, :, :3]
                    overlay_alpha = resized_overlay[:, :, 3] / 255.0
                    
                    overlay_alpha = np.expand_dims(overlay_alpha, axis=2)
                    current_mask = np.expand_dims(current_mask, axis=2)
                    
                    region = result_frame[y_min:y_max+1, x_min:x_max+1]
                    
                    blended = region * (1 - (overlay_alpha * current_mask)) + \
                             overlay_color * (overlay_alpha * current_mask)
                    
                    result_frame[y_min:y_max+1, x_min:x_max+1] = blended.astype(np.uint8)

                result_frame = mask_annotator.annotate(result_frame, detections)
                
                sink.write_frame(result_frame)

        return jsonify({
            "status": "success"
        })
    except Exception as e:
        return jsonify({"error": str(e), "status": "error"}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000)
