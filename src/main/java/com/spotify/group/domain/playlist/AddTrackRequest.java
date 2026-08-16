package com.spotify.group.domain.playlist;

import jakarta.validation.constraints.NotNull;

public record AddTrackRequest(
        @NotNull(message = "El ID de la canción de Deezer es obligatorio")
        Long deezerTrackId
) {}