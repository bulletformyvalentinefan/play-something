package com.spotify.group.domain.track;

import com.spotify.group.domain.track.dto.TrackResponse;
import com.spotify.group.infrastructure.kafka.TrackEventProducer;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/spotify/tracks")
@RequiredArgsConstructor
public class TrackController {

    private final TrackService service;
    private final TrackEventProducer trackEventProducer;

    @GetMapping("/search")
    public ResponseEntity<List<TrackResponse>> searchTracks(@RequestParam String q) {
        List<TrackResponse> results = service.searchTracks(q);
        return ResponseEntity.ok(results);
    }

    @GetMapping("/{trackId}")
    public ResponseEntity<TrackResponse> getTrackById(@PathVariable Long trackId) {
        TrackResponse response = service.getTrackById(trackId);
        return ResponseEntity.ok(response);
    }

    @PostMapping("/{trackId}/play")
    public ResponseEntity<TrackResponse> playTrack(@PathVariable Long trackId, @RequestParam UUID userId) {
        TrackResponse track = service.getTrackById(trackId);
        trackEventProducer.emitTrackPlayed(userId, trackId);
        return ResponseEntity.ok(track);
    }

}
