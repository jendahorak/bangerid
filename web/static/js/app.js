window.spotifyDeviceId = "";

window.onSpotifyWebPlaybackSDKReady = () => {
    const token = window.spotifyToken;
    if (!token) return;

    const player = new Spotify.Player({
        name: 'Bangerid Web Player',
        getOAuthToken: cb => { cb(token); },
        volume: 0.5
    });

    // Ready
    player.addListener('ready', ({ device_id }) => {
        console.log('Ready with Device ID', device_id);
        window.spotifyDeviceId = device_id;
    });

    // Not Ready
    player.addListener('not_ready', ({ device_id }) => {
        console.log('Device ID has gone offline', device_id);
    });

    player.addListener('initialization_error', ({ message }) => {
        console.error('Failed to initialize', message);
    });

    player.addListener('authentication_error', ({ message }) => {
        console.error('Failed to authenticate', message);
    });

    player.addListener('account_error', ({ message }) => {
        console.error('Failed to validate Spotify account', message);
    });

    // State Change Listener (What's playing?)
    player.addListener('player_state_changed', state => {
        if (!state) return;

        const currentTrack = state.track_window.current_track;
        if (!currentTrack) return;

        // Remove playing class from all cards
        document.querySelectorAll('.song-card.is-playing').forEach(el => {
            el.classList.remove('is-playing');
        });

        // Add playing class to current track card
        // Note: Spotify SDK returns IDs sometimes differently (linked tracks), 
        // so this is a best-effort match.
        const trackId = currentTrack.id;
        const card = document.querySelector(`.song-card[data-track-id="${trackId}"]`);
        
        if (card) {
            card.classList.add('is-playing');
            // Optional: Scroll to view if needed, but might be annoying
            // card.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
    });

    player.connect();
};
