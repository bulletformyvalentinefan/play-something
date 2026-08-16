package com.spotify.group.domain.track;

import com.spotify.group.domain.track.dto.DeezerSearchResponse;
import com.spotify.group.domain.track.dto.DeezerTrackResponse;
import com.spotify.group.domain.track.dto.TrackResponse;
import com.spotify.group.infrastructure.client.DeezerClient;
import com.spotify.group.infrastructure.exception.ResourceNotFoundException;
import lombok.RequiredArgsConstructor;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
@RequiredArgsConstructor
public class TrackService {

    private final DeezerClient deezerClient;

    @Cacheable(value = "track_searches", key = "#query")
    public List<TrackResponse> searchTracks(String query) {
        DeezerSearchResponse response = deezerClient.searchTracks(query);
        if (response == null || response.data() == null) {
            return List.of();
        }
        return response.data().stream()
                .map(TrackResponse::fromDeezer)
                .toList();
    }

    @Cacheable(value = "tracks", key = "#trackId")
    public TrackResponse getTrackById(Long trackId) {
        DeezerTrackResponse response = deezerClient.getTrackById(trackId);
        if (response == null || response.id() == null) {
            throw new ResourceNotFoundException("Canción no encontrada en Deezer con ID: " + trackId);
        }
        return TrackResponse.fromDeezer(response);
    }
}