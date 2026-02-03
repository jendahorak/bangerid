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

  window.spotifyPlayer.addListener("player_state_changed", (state) => {
    if (!state || !state.track_window.current_track) return;

    const currentTrack = state.track_window.current_track;

    // 1. Collect all valid URIs for this track
    const activeURIs = new Set();
    activeURIs.add(currentTrack.uri);
    if (currentTrack.linked_from && currentTrack.linked_from.uri) {
      activeURIs.add(currentTrack.linked_from.uri);
    }

    // 2. Find the active card
    let activeCard = null;
    document.querySelectorAll(".song-card").forEach((card) => {
      // Note: use dataset.trackId if the HTML attribute is data-track-id
      if (activeURIs.has(card.dataset.trackId)) {
        activeCard = card;
      }
    });

    // 3. Reset all OTHER cards (not the active one)
    document.querySelectorAll(".song-card.is-playing").forEach((el) => {
      if (el !== activeCard) {
        el.classList.remove("is-playing");
        const btn = el.querySelector(".pause-btn");
        if (btn) btn.classList.remove("is-paused");
      }
    });

    // 4. Update the active card
    if (activeCard) {
      // Remove loading state once SDK confirms playback
      activeCard.classList.remove("is-loading");
      activeCard.classList.add("is-playing");

      // Update pause button based on actual state
      const pauseBtn = activeCard.querySelector(".pause-btn");
      if (pauseBtn) {
        if (state.paused) {
          pauseBtn.classList.add("is-paused");
        } else {
          pauseBtn.classList.remove("is-paused");
        }
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

// Add loading state when HTMX request completes but SDK hasn't confirmed yet
document.body.addEventListener("htmx:afterRequest", (e) => {
  const card = e.detail.elt;
  if (card && card.classList.contains("song-card")) {
    // Add loading state that will be removed when SDK fires player_state_changed
    card.classList.add("is-loading");
  }
});
