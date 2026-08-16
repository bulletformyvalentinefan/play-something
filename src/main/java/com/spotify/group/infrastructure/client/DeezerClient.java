package com.spotify.group.infrastructure.client;

import com.spotify.group.domain.track.dto.DeezerSearchResponse;
import com.spotify.group.domain.track.dto.DeezerTrackResponse;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

@Component
public class DeezerClient {

    private final RestClient restClient;

    public DeezerClient(RestClient.Builder builder) {
        this.restClient = builder
                .baseUrl("https://api.deezer.com")
                .build();
    }

    public DeezerSearchResponse searchTracks(String query) {
        return restClient.get()
                .uri(uriBuilder -> uriBuilder
                        .path("/search")
                        .queryParam("q", query)
                        .build())
                .retrieve()
                .body(DeezerSearchResponse.class);
    }

    public DeezerTrackResponse getTrackById(Long trackId) {
        return restClient.get()
                .uri("/track/{id}", trackId)
                .retrieve()
                .body(DeezerTrackResponse.class);
    }
}