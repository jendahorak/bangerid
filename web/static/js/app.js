window.spotifyDeviceId = "";
window.spotifyPlayer = null;
window.onSpotifyWebPlaybackSDKReady = () => {
  const token = window.spotifyToken;
  if (!token) return;

  window.spotifyPlayer = new Spotify.Player({
    name: "Bangerid Web Player",
    getOAuthToken: (cb) => {
      cb(token);
    },
    volume: 0.5,
  });

  // Ready
  window.spotifyPlayer.addListener("ready", ({ device_id }) => {
    console.log("Ready with Device ID", device_id);
    window.spotifyDeviceId = device_id;
  });

  // Not Ready
  window.spotifyPlayer.addListener("not_ready", ({ device_id }) => {
    console.log("Device ID has gone offline", device_id);
  });

  window.spotifyPlayer.addListener("initialization_error", ({ message }) => {
    console.error("Failed to initialize", message);
  });

  window.spotifyPlayer.addListener("authentication_error", ({ message }) => {
    console.error("Failed to authenticate", message);
  });

  window.spotifyPlayer.addListener("account_error", ({ message }) => {
    console.error("Failed to validate Spotify account", message);
  });

  // State Change Listener (What's playing?)
  window.spotifyPlayer.addListener("player_state_changed", (state) => {
    if (!state) return;

    const currentTrack = state.track_window.current_track;
    if (!currentTrack) return;

    // Remove playing class from all cards
    document.querySelectorAll(".song-card.is-playing").forEach((el) => {
      el.classList.remove("is-playing");
    });

    // Add playing class to current track card
    // Note: Spotify SDK returns IDs sometimes differently (linked tracks),
    // so this is a best-effort match.
    const trackId = currentTrack.id;
    console.log(" Spotify SDK Track ID:", trackId);
    console.log(" Track Name:", currentTrack.name);
    const card = document.querySelector(
      `.song-card[data-track-id="${trackId}"]`,
    );

    if (card) {
      console.log("✅ Found matching card!");
      card.classList.add("is-playing");
      // Optional: Scroll to view if needed, but might be annoying
      // card.scrollIntoView({ behavior: 'smooth', block: 'center' });
    } else {
      console.error("❌ No card found with track ID:", trackId);
      console.log(
        "Available cards:",
        Array.from(document.querySelectorAll(".song-card"))
          .slice(0, 5)
          .map((c) => c.dataset.trackId),
      );
    }

    const pauseBtn = document.querySelector(".song-card.is-playing .pause-btn");
    if (pauseBtn) {
      if (state.paused) {
        pauseBtn.classList.add("playing");
      } else {
        pauseBtn.classList.remove("playing");
      }
    }
  });

  window.spotifyPlayer.connect();
};
document.addEventListener("click", (e) => {
  // Handle playback control buttons
  const button = e.target.closest(".control-btn");

  if (button) {
    if (button.classList.contains("pause-btn")) {
      if (window.spotifyPlayer) {
        window.spotifyPlayer.togglePlay();
      }
      return;
    }

    if (button.classList.contains("next-btn")) {
      const currentCard = button.closest(".song-card");
      const currentIndex = parseInt(currentCard.dataset.index);
      const nextCard = document.querySelector(
        `.song-card[data-index="${currentIndex + 1}"]`,
      );

      if (nextCard) {
        nextCard.click();
      }
      return;
    }

    if (button.classList.contains("prev-btn")) {
      const currentCard = button.closest(".song-card");
      const currentIndex = parseInt(currentCard.dataset.index);
      const prevCard = document.querySelector(
        `.song-card[data-index="${currentIndex - 1}"]`,
      );

      if (prevCard) {
        prevCard.click();
      }
      return;
    }
  }
});
