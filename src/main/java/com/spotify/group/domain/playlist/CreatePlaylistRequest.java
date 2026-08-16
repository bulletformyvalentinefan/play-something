package com.spotify.group.domain.playlist;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import java.util.UUID;

public record CreatePlaylistRequest(
        @NotNull(message = "El ID de usuario es obligatorio")
        UUID userId,

        @NotBlank(message = "El título no puede estar vacío")
        String titulo,

        String descripcion,

        Boolean esPublica
) {}