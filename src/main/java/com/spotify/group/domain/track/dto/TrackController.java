package com.spotify.group.domain.track.dto;

import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Controller;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;

import java.util.List;

@Controller
@RequestMapping("/api/v1/spotify/tracks")
@RequiredArgsConstructor
public class TrackController {

    private final TrackService service;

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

}
