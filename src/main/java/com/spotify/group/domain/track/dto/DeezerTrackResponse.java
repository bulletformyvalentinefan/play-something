package com.spotify.group.domain.track.dto;

import com.fasterxml.jackson.annotation.JsonProperty;

public record DeezerTrackResponse(
        Long id,
        String title,
        Integer duration,
        String preview,
        ArtistDTO artist,
        AlbumDTO album
) {
    public record ArtistDTO(Long id, String name) {}
    public record AlbumDTO(Long id, String title, @JsonProperty("cover_medium") String coverMedium) {}
}