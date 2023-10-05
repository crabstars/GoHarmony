<script>
	import Youtube from '$lib/youtube.svelte';
	import { onMount, onDestroy, tick } from 'svelte';
	import { v4 as uuidv4 } from 'uuid';

	let initialVideoId;
	const PUBLIC_BASE_URL = 'https://askpaul.duckdns.org/api/';
	onMount(async () => {
		// 1. get current state
		try {
			const response = await fetch(PUBLIC_BASE_URL + 'current-state', {
				method: 'GET',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				throw new Error('Network response was not ok');
			}
			videoState = await response.json();
			initialVideoId = videoState.video_link.replace('https://www.youtube.com/watch?v=', '');

			// setTimeout(changeVideoByEvent(videoState.video_link), 1000);
			// changeVideo();
			// // jumpToSeconds(videoState.video_timestamp);
			// // togglePlayPauseFromEvent(videoState.video_running);
		} catch (error) {
			console.error('There was a problem with the fetch operation:', error);
		}

		// 2. sub to event
		const eventSource = new EventSource(PUBLIC_BASE_URL + '/events/' + guid);

		eventSource.onmessage = (event) => {
			const data = JSON.parse(event.data);
			console.log('got new event');
			console.log(data);

			if (data['etag'] > videoState.etag) {
				videoState.etag = data['etag'];
				// change that i have to edit video_running and isPlaying
				togglePlayPauseFromEvent(data['video_running']);

				changeVideoByEvent(data['video_link']);

				if (Math.abs(data['video_timestamp'] - player.getCurrentTime()) > 2) {
					videoState.video_timestamp = data['video_timestamp'];
					jumpToSeconds(videoState.video_timestamp);
				}
			}
		};

		eventSource.onerror = (error) => {
			console.error('EventSource failed:', error);
			eventSource.close();
		};

		eventSource.onopen = () => {
			console.log('connection open');
		};

		//3. check timestamp
		checkInterval = setInterval(() => {
			if (player) {
				if (videoState.video_running) {
					videoState.video_timestamp += 1;
				}
				const currentTime = Math.round(player.getCurrentTime());
				if (Math.abs(currentTime - videoState.video_timestamp) > 2) {
					handleSeeked();
				}
			}
		}, 1000); // Check every second.

		checkVideoLoaded = setInterval(() => {
			if (player && player.getPlayerState === undefined) {
				console.log('No video is loaded, loading initial video');

				// load an initial video
				player.loadVideoById(videoState.video_link.replace('https://www.youtube.com/watch?v=', ''));
			}
		}, 1000);
		return () => {
			console.log('connection close');
			eventSource.close();
		};
	});

	onDestroy(() => {
		clearInterval(checkInterval);
		clearInterval(checkVideoLoaded);
	});

	const guid = uuidv4();
	let videoState = {
		video_link: 'https://www.youtube.com/watch?v=p7DrHGrpqFU',
		video_running: false,
		video_timestamp: 0,
		request_timestamp: 0,
		etag: 0
	};

	let checkInterval;
	let checkVideoLoaded;
	let player;

	const changeVideo = () => {
		console.log('changing video link');
		player.loadVideoById(videoState.video_link.replace('https://www.youtube.com/watch?v=', ''));
		videoState.video_timestamp = 0;
		updateVideoState();
	};

	const changeVideoByEvent = (event_video_link) => {
		if (event_video_link === videoState.video_link) {
			console.log('same video link');
			return;
		}
		console.log('changing video link by event');
		videoState.video_link = event_video_link;
		videoState.video_timestamp = 0;
		player.loadVideoById(event_video_link.replace('https://www.youtube.com/watch?v=', ''));
	};

	const togglePlayPauseFromEvent = (palying) => {
		if (palying === videoState.video_running) {
			console.log('return');
			return;
		}
		if (palying === false) {
			console.log('pause');
			player.pauseVideo();
		} else {
			console.log('play');
			player.playVideo();
		}
		videoState.video_running = palying;
	};

	const jumpToSeconds = (seconds) => {
		if (!isNaN(seconds)) player.seekTo(seconds, true);
	};

	const handleStateChange = (palying) => {
		console.log('playing');
		if (videoState.video_running != palying) {
			videoState.video_running = palying;
			updateVideoState();
		}
	};

	const handleSeeked = () => {
		videoState.video_timestamp = Math.round(player.getCurrentTime());
		updateVideoState();
	};

	function handlePlayerMount() {
		if (player) {
			console.log('Player mount');
			player.loadVideoById(videoState.video_link.replace('https://www.youtube.com/watch?v=', ''));
		}
	}

	async function updateVideoState() {
		try {
			const response = await fetch(PUBLIC_BASE_URL + 'change-state/' + guid, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(videoState)
			});

			if (!response.ok) throw new Error(`HTTP error! Status: ${response.status}`);

			console.log('Video state updated successfully');
		} catch (error) {
			console.error('Failed to update video state', error);
		}
	}
</script>

<div class="youtube-container">
	<Youtube
		bind:player
		{initialVideoId}
		on:playing={() => handleStateChange(true)}
		on:paused={() => handleStateChange(false)}
		on:playerMount={() => handlePlayerMount()}
	/>
</div>

<div>
	<label for="videoLink">Video Link: </label>
	<input
		id="videoLink"
		type="text"
		bind:value={videoState.video_link}
		placeholder="Enter Video Link"
	/>
	<button on:click={changeVideo}>change video</button>
</div>

<style>
	/* thanks chatgpt */
	:global(body) {
		background-color: #333333; /* Dark Grey Background */
		color: white; /* Set Text color to white for better readability in dark mode */
	}

	.youtube-container {
		width: 80vw; /* Or any specific width */
		height: 45vw; /* Maintain aspect ratio, height = 0.5625 * width for 16:9 videos */
		max-width: 1280px; /* Maximum width you want */
		max-height: 720px; /* Maximum height you want */
		margin: 0 auto; /* Centering: auto margin on left and right */
		position: relative; /* Keep this as relative so if you use absolute positioning within the Youtube component, it will be relative to this container */
		background-color: #333333; /* Dark Grey Background */
		color: white; /* Set Text color to white (or any lighter shade) for better readability in dark mode */
	}
</style>
