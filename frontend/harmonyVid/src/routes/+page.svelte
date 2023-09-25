<script>
	import Youtube from '$lib/youtube.svelte';
	import { onMount, onDestroy } from 'svelte';
	import { v4 as uuidv4 } from 'uuid';

	onMount(async () => {
		// 1. get current state
		try {
			console.log(guid);
			const response = await fetch('http://localhost:3000/current-state', {
				method: 'GET',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				throw new Error('Network response was not ok');
			}
			videoState = await response.json();
		} catch (error) {
			console.error('There was a problem with the fetch operation:', error);
		}

		// 2. sub to event
		const eventSource = new EventSource('http://localhost:3000/events/' + guid);

		eventSource.onmessage = (event) => {
			const data = JSON.parse(event.data);
			if (data['video_running'] != isPlaying) {
				togglePlayPause();
			}

			if (Math.abs(data['video_timestamp'] - player.getCurrentTime()) > 2) {
				videoState.video_timestamp = data['video_timestamp'];
				jumpToSeconds(videoState.video_timestamp);
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
			if (player && player.getCurrentTime) {
				if (YT.PlayerState.PLAYING) {
					videoState.video_timestamp += 1;
				}
				const currentTime = Math.round(player.getCurrentTime());
				console.log(currentTime);
				if (Math.abs(currentTime - videoState.video_timestamp) > 2) {
					handleSeeked();
				}
			}
		}, 1000); // Check every second.

		return () => {
			console.log('connection close');
			eventSource.close(); // Close the connection when the component is destroyed
		};
	});

	onDestroy(() => {
		clearInterval(checkInterval); // Clear the interval when the component is destroyed
	});

	const guid = uuidv4();
	let videoState = {
		video_id: '',
		video_running: false,
		video_timestamp: 0,
		request_timestamp: 0
	};

	let checkInterval;
	let player;
	let isPlaying = false;
	let videoId = 'FnLvyysSCw4'; // default
	let logInterval;

	const changeVideo = () => {
		console.log('changing video id');
		player.loadVideoById(videoId);
	};

	const togglePlayPause = () => {
		if (isPlaying) {
			player.pauseVideo();
			clearInterval(logInterval); // clear interval when video is paused
		} else {
			player.playVideo();
			startLogging(); // start logging when video is played
		}
		isPlaying = !isPlaying;
		videoState.video_running = isPlaying;
		updateVideoState();
	};

	const jumpToSeconds = (seconds) => {
		if (!isNaN(seconds)) player.seekTo(seconds, true);
	};

	const startLogging = () => {
		logInterval = setInterval(() => {
			const currentTime = player.getCurrentTime();
			//console.log(`Current Time: ${currentTime}`);
		}, 1000);
	};

	const handleStateChange = (isPlaying) => {
		if (isPlaying != videoState.video_running) {
			videoState.video_running = isPlaying;
			updateVideoState();
		}
	};

	const handleSeeked = () => {
		videoState.video_timestamp = Math.round(player.getCurrentTime());
		updateVideoState();
	};

	async function updateVideoState() {
		// TODO right now if we get a change from the server we also send a patch to the server because the onPlayerStateChange is triggerd if we change isPlaying
		// => fix soon bec we can get a infinite loop to send request again and again or backend dont send state to same user again => better
		try {
			const response = await fetch('http://localhost:3000/change-state/' + guid, {
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

<Youtube
	bind:player
	on:playing={() => handleStateChange(true)}
	on:paused={() => handleStateChange(false)}
/>

<div>
	<label for="videoId">Video ID: </label>
	<input id="videoId" type="text" bind:value={videoId} placeholder="Enter Video ID" />
	<button on:click={changeVideo}>change video</button>
</div>
