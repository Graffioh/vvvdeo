class VideoPlayer {
  constructor() {
    this.player = null;
    this.initApp();
  }

  initApp() {
    // Install built-in polyfills to patch browser incompatibilities.
    shaka.polyfill.installAll();

    // Check to see if the browser supports the basic APIs Shaka needs.
    if (shaka.Player.isBrowserSupported()) {
      // Everything looks good!
      this.initPlayer();
    } else {
      // This browser does not have the minimum set of APIs we need.
      console.error("Browser not supported!");
    }
  }

  async initPlayer() {
    // Create a Player instance.
    const video = document.getElementById("shaka-player-video");
    this.player = new shaka.Player();
    await this.player.attach(video);

    // Attach player to the window to make it easy to access in the JS console.
    window.player = this.player;

    // Listen for error events.
    this.player.addEventListener("error", this.onErrorEvent);
  }

  async loadManifest(manifestUri) {
    // Try to load a manifest.
    // This is an asynchronous process.
    try {
      await this.player.load(manifestUri);
      // This runs if the asynchronous load is successful.
      console.log("The video has now been loaded!");
    } catch (e) {
      // onError is executed if the asynchronous load fails.
      this.onError(e);
    }
  }

  onErrorEvent(event) {
    // Extract the shaka.util.Error object from the event.
    this.onError(event.detail);
  }

  onError(error) {
    // Log the error.
    console.error("Error code", error.code, "object", error);
  }

  setVideoPlayerVisible() {
    // Set video visibility in the frontend
    const video = document.getElementById("shaka-player-video");
    video.hidden = false;
  }
}

export { VideoPlayer };
