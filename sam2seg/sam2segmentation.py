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

def clear_directory(directory_path):
    for filename in os.listdir(directory_path):
        file_path = os.path.join(directory_path, filename)
        if os.path.isfile(file_path):
            os.remove(file_path)
        elif os.path.isdir(file_path):
            shutil.rmtree(file_path)

clear_directory("./static/")

app = Flask(__name__)

# Setup device and model
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
sam2_checkpoint = "./src/sam-2/checkpoints/sam2.1_hiera_tiny.pt"
model_cfg = "./configs/sam2.1/sam2.1_hiera_t.yaml"
predictor = build_sam2_video_predictor(model_cfg, sam2_checkpoint, device=device)

video_dir = "./frames"

@app.route("/predict", methods=["POST"])
def predict():
    inference_state = predictor.init_state(video_path=video_dir)
    predictor.reset_state(inference_state)

    try:
        data = request.get_json()
        print("DATA:")
        print(data)

        points = data.get("coordinates", [])
        labels = data.get("labels", [])
        print("POINTS:")
        print(points)
        print("LABELS:")
        print(labels)

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
    print("YOOOOOOOOOOOOO")

    inference_state = predictor.init_state(video_path=video_dir)
    predictor.reset_state(inference_state)

    try:
        # Parse points from request
        data = request.get_json()
        print("DATA:")
        print(data)

        points = data.get("coordinates", [])
        labels = data.get("labels", [])
        print("POINTS:")
        print(points)
        print("LABELS:")
        print(labels)

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

        frame_path = os.path.join(video_dir, f"{ann_frame_idx:05d}.jpg")
        if not os.path.exists(frame_path):
            return jsonify({"error": f"Frame not found: {frame_path}"}), 404

        frame_image = Image.open(frame_path).convert("RGBA")
        mask_image_rgb = (mask_image[..., :3] * 255).astype(np.uint8)  

        video_segments = {}  
        for out_frame_idx, out_obj_ids, out_mask_logits2 in predictor.propagate_in_video(inference_state):
            print("+++++++++++++")
            video_segments[out_frame_idx] = {
                out_obj_id: (out_mask_logits2[i] > 0.0).cpu().numpy()
                for i, out_obj_id in enumerate(out_obj_ids)
            }

        print("00000000000")

        output_dir = "./static"
        os.makedirs(output_dir, exist_ok=True)

        frame_names = [
            p for p in os.listdir(video_dir)
            if os.path.splitext(p)[-1] in [".jpg", ".jpeg", ".JPG", ".JPEG", ".png"]
        ]
        frame_names.sort(key=lambda p: int(os.path.splitext(p)[0]))

        segmented_image_paths = []
        vis_frame_stride = 5
        for out_frame_idx in range(0, len(frame_names), vis_frame_stride):
            print("111111111111")
            for out_obj_id, out_mask in video_segments[out_frame_idx].items():
                print("22222")
                mask_data = out_mask
                color = np.concatenate([np.random.random(3), np.array([0.6])], axis=0)  
                h, w = mask_data.shape[-2:]
                mask_image = mask_data.reshape(h, w, 1) * color.reshape(1, 1, -1)
                mask_image_rgb = (mask_image[..., :3] * 255).astype(np.uint8)  
                segmented_image = Image.fromarray(mask_image_rgb, 'RGB')

                timestamp = str(time.time_ns())
                segmented_image_path = os.path.join(output_dir, f"segmented_frame_{timestamp}.png")
                print(segmented_image_path)
                segmented_image.save(segmented_image_path)
                segmented_image_paths.append(segmented_image_path)

        return jsonify({
            "status": "success",
            "segmented_image_paths": segmented_image_paths
        })

    except Exception as e:
        return jsonify({"error": str(e), "status": "error"}), 500

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000)
