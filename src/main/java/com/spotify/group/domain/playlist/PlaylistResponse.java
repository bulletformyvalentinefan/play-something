package com.spotify.group.domain.playlist;

import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

public record PlaylistResponse(
        UUID id,
        String titulo,
        String descripcion,
        Boolean esPublica,
        UUID userId,
        LocalDateTime fechaCreacion,
        List<Long> trackIds
) {
    public static PlaylistResponse fromEntity(Playlist playlist) {
        List<Long> trackIds = playlist.getTracks() != null
                ? playlist.getTracks().stream().map(PlaylistTrack::getDeezerTrackId).toList()
                : List.of();

        return new PlaylistResponse(
                playlist.getId(),
                playlist.getTitulo(),
                playlist.getDescripcion(),
                playlist.getEsPublica(),
                playlist.getUser().getId(),
                playlist.getFechaCreacion(),
                trackIds
        );
    }
}