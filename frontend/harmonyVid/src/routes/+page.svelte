<script>
    import Youtube from '$lib/youtube.svelte';
  
    let player;
    // TODO when on Mount make a request to the server to get all infromation
    let isPlaying = false;
    let timestamp = '';
    let videoId = 'FnLvyysSCw4'; // default
    let logInterval; // to store the ID of the interval
  
    const changeVideo = () => {
      console.log('changing video id');
      player.loadVideoById(videoId);
    }
  
    const togglePlayPause = () => {
      if (isPlaying) {
        player.pauseVideo();
        clearInterval(logInterval); // clear interval when video is paused
      } else {
        player.playVideo();
        startLogging(); // start logging when video is played
      }
      isPlaying = !isPlaying;
    }
  
    const handlePlaying = () => {
      console.log('Video is playing');
      isPlaying = true;
      startLogging(); // start logging when video is played
    }
    
    const handlePaused = () => {
      console.log('Video is paused');
      isPlaying = false;
      clearInterval(logInterval); // clear interval when video is paused
    }
    
    const jumpToTimestamp = () => {
      const seconds = parseTimestamp(timestamp);
      if (!isNaN(seconds)) player.seekTo(seconds, true);
    }
    
    const parseTimestamp = (timestamp) => {
      const parts = timestamp.split(':').reverse();
      let seconds = 0;
      parts.forEach((part, index) => {
        seconds += Number(part) * Math.pow(60, index);
      });
      return seconds;
    }
    
    const startLogging = () => {
      logInterval = setInterval(() => {
        const currentTime = player.getCurrentTime();
        console.log(`Current Time: ${currentTime}`);
      }, 1000);
    }
</script>
  
<Youtube bind:player on:playing={handlePlaying} on:paused={handlePaused}/>
  
<div>
    <label for="videoId">Video ID: </label>
    <input id="videoId" type="text" bind:value={videoId} placeholder="Enter Video ID"/>
    <button on:click={changeVideo}>change video</button>
</div>
<button on:click={togglePlayPause}>{isPlaying ? 'Pause' : 'Play'}</button>
  
<div>
    <label for="timestamp">Timestamp: </label>
    <input id="timestamp" type="text" bind:value={timestamp} placeholder="mm:ss"/>
    <button on:click={jumpToTimestamp}>Jump to Timestamp</button>
</div>
