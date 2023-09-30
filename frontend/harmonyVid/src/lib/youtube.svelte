<script>
	import { onMount, createEventDispatcher } from 'svelte';

	export let player;
	export let initialVideoId = '';

	const ytPlayerId = 'youtube-player';
	const dispatch = createEventDispatcher();

	onMount(() => {
		function load() {
			player = new YT.Player(ytPlayerId, {
				height: '100%',
				width: '100%',
				videoId: initialVideoId,
				playerVars: { autoplay: 1 },
				events: {
					onStateChange: onPlayerStateChange
				}
			});
			dispatch('playerMount');
		}
		// TODO each time the player.time changes we should update videoState

		function onPlayerStateChange(event) {
			switch (event.data) {
				case YT.PlayerState.PLAYING:
					dispatch('playing');
					break;
				case YT.PlayerState.PAUSED:
					dispatch('paused');
					break;
				// handle other states if needed like buffered
			}
		}

		if (window.YT) {
			load();
		} else {
			window.onYouTubeIframeAPIReady = load;
		}
	});
</script>

<svelte:head>
	<script src="https://www.youtube.com/iframe_api"></script>
</svelte:head>

<div id={ytPlayerId} />
