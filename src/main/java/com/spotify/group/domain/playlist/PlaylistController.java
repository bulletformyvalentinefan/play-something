package com.spotify.group.domain.playlist;

import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

@RestController
@RequestMapping("/api/v1/spotify")
@RequiredArgsConstructor
public class PlaylistController {

    private final PlaylistService playlistService;

    @PostMapping("/playlists")
    public ResponseEntity<PlaylistResponse> createPlaylist(@Valid @RequestBody CreatePlaylistRequest request) {
        PlaylistResponse response = playlistService.createPlaylist(request);
        return ResponseEntity.status(HttpStatus.CREATED).body(response);
    }

    @GetMapping("/users/{userId}/playlists")
    public ResponseEntity<List<PlaylistResponse>> getUserPlaylists(@PathVariable UUID userId) {
        return ResponseEntity.ok(playlistService.getPlaylistsByUserId(userId));
    }

    @GetMapping("/playlists/{playlistId}")
    public ResponseEntity<PlaylistResponse> getPlaylistById(@PathVariable UUID playlistId) {
        return ResponseEntity.ok(playlistService.getPlaylistById(playlistId));
    }

    @PostMapping("/playlists/{playlistId}/tracks")
    public ResponseEntity<Void> addTrack(@PathVariable UUID playlistId, @Valid @RequestBody AddTrackRequest request) {
        playlistService.addTrackToPlaylist(playlistId, request.deezerTrackId());
        return ResponseEntity.status(HttpStatus.CREATED).build();
    }

    @DeleteMapping("/playlists/{playlistId}/tracks/{trackId}")
    public ResponseEntity<Void> removeTrack(@PathVariable UUID playlistId, @PathVariable Long trackId) {
        playlistService.removeTrackFromPlaylist(playlistId, trackId);
        return ResponseEntity.noContent().build();
    }
}