package com.spotify.group.domain.track.dto;

public record TrackResponse(
        Long id,
        String title,
        Integer duration,
        String previewUrl,
        String artistName,
        String albumCover
) {
    public static TrackResponse fromDeezer(DeezerTrackResponse deezer) {
        return new TrackResponse(
                deezer.id(),
                deezer.title(),
                deezer.duration(),
                deezer.preview(),
                deezer.artist() != null ? deezer.artist().name() : null,
                deezer.album() != null ? deezer.album().coverMedium() : null
        );
    }
}