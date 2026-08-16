package com.spotify.group.domain.track.dto;

import com.spotify.group.infrastructure.client.DeezerClient;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
@RequiredArgsConstructor
public class TrackService {

    private final DeezerClient deezerClient;

    public List<TrackResponse> searchTracks(String query) {
        DeezerSearchResponse response = deezerClient.searchTracks(query);
        if (response == null || response.data() == null) {
            return List.of();
        }
        return response.data().stream().map(TrackResponse::fromDeezer).toList();

    }
    public TrackResponse getTrackById(Long trackId) {
        DeezerTrackResponse response = deezerClient.getTrackById(trackId);
        if (response == null || response.id() == null) {
            throw new IllegalArgumentException("Canción no encontrada en Deezer con ID: " + trackId);
        }
        return TrackResponse.fromDeezer(response);
    }
}
